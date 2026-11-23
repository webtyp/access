package ui

import (
	"webtyp.com/access"
	"webtyp.com/router"
	"webtyp.com/view"
)

// Screens declares the screens access offers to the app's menu.
//
//	screens := ui.Screens(caller)
func Screens(caller router.Caller) []view.Screen {
	return []view.Screen{{
		ID: ID, Label: LabelGroup, Icon: string(IconPeople), Help: helpGroup,
		Children: []view.Screen{
			{ID: RolesID, Label: LabelRoles, Icon: string(IconRoles), Permission: "access_role",
				Presenter: access.NewRolesView(caller, view.WithDescription(descRoles))},
			{ID: AssignmentsID, Label: LabelAssignments, Icon: string(IconAssignments), Permission: "access_assignment",
				Main: NewAssignmentsView(caller), Help: descAssignments},
			{ID: AuditID, Label: LabelAudit, Icon: string(IconAudit), Permission: "access_audit",
				Presenter: access.NewAuditView(caller, view.WithDescription(descAudit))},
		},
	}}
}
