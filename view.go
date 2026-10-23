package access

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

const (
	TitleRoles       = "Roles"
	TitleAssignments = "Assignments"
	TitleAudit       = "Audit"
)

// Item projects a RoleRecord as a view.Item (view.Itemizer).
func (r *RoleRecord) Item() view.Item {
	return view.Item{
		ID:          r.Id,
		Label:       r.Name,
		Description: r.Code,
	}
}

// Item projects an Assignment as a view.Item (view.Itemizer).
func (a *Assignment) Item() view.Item {
	desc := a.RoleCode
	if a.Name != "" {
		desc = a.Name + " (" + a.RoleCode + ")"
	}
	return view.Item{
		ID:          a.Id,
		Label:       a.Email,
		Description: desc,
	}
}

// Item projects an AuditEntry as a view.Item (view.Itemizer).
func (e *AuditEntry) Item() view.Item {
	desc := e.Action + " " + e.Target
	if e.Detail != "" {
		desc += " — " + e.Detail
	}
	return view.Item{
		ID:          e.Id,
		Label:       e.ActorEmail,
		Description: desc,
	}
}

// NewRolesView builds the Presenter for the roles CRUD screen.
func NewRolesView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{
			Module: ModelName,
			List:   OpListRoles,
			Save:   OpSaveRoles,
			Delete: OpDeleteRoles,
		},
		func() model.ModelSlice { return &RoleRecordList{} },
	)
	return view.New(b, &RoleRecord{}, view.WithTitle(TitleRoles))
}

// NewAssignmentsView builds the Presenter for the assignments CRUD screen.
func NewAssignmentsView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{
			Module: ModelName,
			List:   OpListAssignments,
			Save:   OpAssignRoles,
			Delete: OpRevokeAssignments,
		},
		func() model.ModelSlice { return &AssignmentList{} },
	)
	return view.New(b, &Assignment{}, view.WithTitle(TitleAssignments))
}

// NewAuditView builds the Presenter for the audit trail view (read-only list).
func NewAuditView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{
			Module: ModelName,
			List:   OpListAudit,
		},
		func() model.ModelSlice { return &AuditEntryList{} },
	)
	return view.New(b, &AuditEntry{}, view.WithTitle(TitleAudit))
}
