package ui

import (
	"webtyp.com/access"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

// RolesBrowser builds the roles management UI module.
func RolesBrowser(caller router.Caller, ids model.IDGenerator) (platformd.UIModule, error) {
	v, err := crudview.New(crudview.Config{
		ParentID:  ID + ".roles",
		Presenter: access.NewRolesView(caller),
		IDs:       ids,
		NewRecord: func() model.Model {
			return &access.RoleRecord{}
		},
	})
	if err != nil {
		return nil, err
	}
	return platformd.NewUIModule(ID+".roles", LabelRoles, IconRoles, v), nil
}

// AssignmentsBrowser builds the role assignments UI module.
func AssignmentsBrowser(caller router.Caller, ids model.IDGenerator) (platformd.UIModule, error) {
	v, err := crudview.New(crudview.Config{
		ParentID:  ID + ".assignments",
		Presenter: access.NewAssignmentsView(caller),
		IDs:       ids,
		NewRecord: func() model.Model {
			return &access.Assignment{}
		},
	})
	if err != nil {
		return nil, err
	}

	return platformd.NewUIModule(ID+".assignments", LabelAssignments, IconAssignments, v), nil
}

// AuditBrowser builds the audit trail UI module (read-only list).
func AuditBrowser(caller router.Caller, ids model.IDGenerator) (platformd.UIModule, error) {
	v, err := crudview.New(crudview.Config{
		ParentID:  ID + ".audit",
		Presenter: access.NewAuditView(caller),
		IDs:       ids,
	})
	if err != nil {
		return nil, err
	}
	return platformd.NewUIModule(ID+".audit", LabelAudit, IconAudit, v), nil
}

// Browser builds all three access administration screens in standard order:
// Roles, Assignments, Audit.
func Browser(caller router.Caller, ids model.IDGenerator) ([]platformd.UIModule, error) {
	caller = view.NewRecords().Caller(caller)
	rolesMod, err := RolesBrowser(caller, ids)
	if err != nil {
		return nil, err
	}
	assignMod, err := AssignmentsBrowser(caller, ids)
	if err != nil {
		return nil, err
	}
	auditMod, err := AuditBrowser(caller, ids)
	if err != nil {
		return nil, err
	}
	return []platformd.UIModule{rolesMod, assignMod, auditMod}, nil
}
