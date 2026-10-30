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
			{ID: ID + ".people", Label: LabelPeople, Icon: string(IconPeople), Permission: access.ResourceAssignment,
				Presenter: access.NewPeopleView(caller, view.WithDescription(descPeople))},
			{ID: ID + ".roles", Label: LabelRoles, Icon: string(IconRoles), Permission: access.ResourceRole,
				Presenter: access.NewRolesView(caller, view.WithDescription(descRoles))},
			{ID: ID + ".assignments", Label: LabelAssignments, Icon: string(IconAssignments), Permission: access.ResourceAssignment,
				Presenter: access.NewAssignmentsView(caller, view.WithDescription(descAssignments))},
			{ID: ID + ".audit", Label: LabelAudit, Icon: string(IconAudit), Permission: access.ResourceAudit,
				Presenter: access.NewAuditView(caller, view.WithDescription(descAudit))},
		},
	}}
}
