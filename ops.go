package access

import (
	"webtyp.com/model"
	"webtyp.com/router"
)

// Operation names (qualified on the wire as "access.<op>").
const (
	OpListRoles         = "list_roles"
	OpSaveRoles         = "save_roles"
	OpDeleteRoles       = "delete_roles"
	OpListAssignments   = "list_assignments"
	OpAssignRoles       = "assign_roles"
	OpRevokeAssignments = "revoke_assignments"
	OpListAudit         = "list_audit"
)

// RBAC resources the operations require. The app grants them to its
// administrator role.
const (
	ResourceRole       model.Resource = "access_role"
	ResourceAssignment model.Resource = "access_assignment"
	ResourceAudit      model.Resource = "access_audit"
)

// Resources lists every resource above, for the app's permission list.
//
// Example: resources = append(resources, access.Resources()...)
func Resources() []model.Resource {
	return []model.Resource{ResourceRole, ResourceAssignment, ResourceAudit}
}

// MountOperations registers the seven operations. With AssuranceMultiFactor
// every one also RequiresMFA.
func (m *Module) MountOperations(reg router.OperationRegistry) {
	m.gate(reg.Operation(OpListRoles, m.opListRoles).Requires(ResourceRole, model.Read))
	m.gate(reg.Operation(OpSaveRoles, m.opSaveRoles).Requires(ResourceRole, model.Create|model.Update).Accepts(&SaveRolesArgs{}))
	m.gate(reg.Operation(OpDeleteRoles, m.opDeleteRoles).Requires(ResourceRole, model.Delete).Accepts(&IDsArgs{}))
	m.gate(reg.Operation(OpListAssignments, m.opListAssignments).Requires(ResourceAssignment, model.Read))
	m.gate(reg.Operation(OpAssignRoles, m.opAssignRoles).Requires(ResourceAssignment, model.Create).Accepts(&AssignRolesArgs{}))
	m.gate(reg.Operation(OpRevokeAssignments, m.opRevokeAssignments).Requires(ResourceAssignment, model.Delete).Accepts(&IDsArgs{}))
	m.gate(reg.Operation(OpListAudit, m.opListAudit).Requires(ResourceAudit, model.Read))
}

func (m *Module) gate(r router.Route) {
	if m.assurance == model.AssuranceMultiFactor {
		r.RequiresMFA()
	}
}

type SaveRolesArgs struct {
	Records []RoleRecord
}

func (a *SaveRolesArgs) ModelName() string { return "save_roles_args" }
func (a *SaveRolesArgs) Schema() []model.Field { return nil }
func (a *SaveRolesArgs) Pointers() []any { return nil }
func (a *SaveRolesArgs) IsNil() bool { return a == nil }
func (a *SaveRolesArgs) EncodeFields(w model.FieldWriter) {
	arr := w.Array("records", len(a.Records))
	for i := range a.Records {
		arr.Object(&a.Records[i])
	}
}
func (a *SaveRolesArgs) DecodeFields(r model.FieldReader) {
	arr, ok := r.Array("records")
	if !ok {
		return
	}
	n := arr.Len()
	for i := 0; i < n; i++ {
		var rec RoleRecord
		arr.Object(i, &rec)
		a.Records = append(a.Records, rec)
	}
}

type AssignRolesArgs struct {
	Records []Assignment
}

func (a *AssignRolesArgs) ModelName() string { return "assign_roles_args" }
func (a *AssignRolesArgs) Schema() []model.Field { return nil }
func (a *AssignRolesArgs) Pointers() []any { return nil }
func (a *AssignRolesArgs) IsNil() bool { return a == nil }
func (a *AssignRolesArgs) EncodeFields(w model.FieldWriter) {
	arr := w.Array("records", len(a.Records))
	for i := range a.Records {
		arr.Object(&a.Records[i])
	}
}
func (a *AssignRolesArgs) DecodeFields(r model.FieldReader) {
	arr, ok := r.Array("records")
	if !ok {
		return
	}
	n := arr.Len()
	for i := 0; i < n; i++ {
		var rec Assignment
		arr.Object(i, &rec)
		a.Records = append(a.Records, rec)
	}
}

type IDsArgs struct {
	Ids []string
}

func (a *IDsArgs) ModelName() string { return "ids_args" }
func (a *IDsArgs) Schema() []model.Field { return nil }
func (a *IDsArgs) Pointers() []any { return nil }
func (a *IDsArgs) IsNil() bool { return a == nil }
func (a *IDsArgs) EncodeFields(w model.FieldWriter) {
	arr := w.Array("ids", len(a.Ids))
	for _, id := range a.Ids {
		arr.String(id)
	}
}
func (a *IDsArgs) DecodeFields(r model.FieldReader) {
	arr, ok := r.Array("ids")
	if !ok {
		return
	}
	n := arr.Len()
	for i := 0; i < n; i++ {
		id := arr.String(i)
		if id != "" {
			a.Ids = append(a.Ids, id)
		}
	}
}

func (m *Module) opListRoles(ctx router.Context) {
	list, err := m.ListRoles()
	if err != nil {
		writeError(ctx, err)
		return
	}
	var res RoleRecordList
	for i := range list {
		res = append(res, &list[i])
	}
	ctx.Encode(&res)
}

func (m *Module) opSaveRoles(ctx router.Context) {
	var args SaveRolesArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Records) == 0 {
		writeError(ctx, ErrNoRecords)
		return
	}
	actor := ctx.UserID()
	var last RoleRecord
	for _, rec := range args.Records {
		if err := m.SaveRole(actor, rec); err != nil {
			writeError(ctx, err)
			return
		}
		last = rec
	}
	ctx.Encode(&last)
}

func (m *Module) opDeleteRoles(ctx router.Context) {
	var args IDsArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Ids) == 0 {
		writeError(ctx, ErrNoIds)
		return
	}
	actor := ctx.UserID()
	for _, id := range args.Ids {
		if err := m.DeleteRole(actor, id); err != nil {
			writeError(ctx, err)
			return
		}
	}
	ctx.WriteStatus(200)
}

func (m *Module) opListAssignments(ctx router.Context) {
	list, err := m.ListAssignments()
	if err != nil {
		writeError(ctx, err)
		return
	}
	var res AssignmentList
	for i := range list {
		res = append(res, &list[i])
	}
	ctx.Encode(&res)
}

func (m *Module) opAssignRoles(ctx router.Context) {
	var args AssignRolesArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Records) == 0 {
		writeError(ctx, ErrNoRecords)
		return
	}
	actor := ctx.UserID()
	var last Assignment
	for _, rec := range args.Records {
		a, err := m.Assign(actor, rec.Email, rec.RoleCode)
		if err != nil {
			writeError(ctx, err)
			return
		}
		last = a
	}
	ctx.Encode(&last)
}

func (m *Module) opRevokeAssignments(ctx router.Context) {
	var args IDsArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Ids) == 0 {
		writeError(ctx, ErrNoIds)
		return
	}
	actor := ctx.UserID()
	for _, id := range args.Ids {
		if err := m.Revoke(actor, id); err != nil {
			writeError(ctx, err)
			return
		}
	}
	ctx.WriteStatus(200)
}

func (m *Module) opListAudit(ctx router.Context) {
	list, err := m.ListAudit()
	if err != nil {
		writeError(ctx, err)
		return
	}
	var res AuditEntryList
	for i := range list {
		res = append(res, &list[i])
	}
	ctx.Encode(&res)
}
