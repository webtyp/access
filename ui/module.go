package ui

import (
	"webtyp.com/lang"
	"webtyp.com/svg"
)

// ID is this module's root identity on client nav rails.
const ID = "access"

// Labels for navigation rail and headers.
const (
	LabelRoles       = "Roles"
	LabelAssignments = "Assignments"
	LabelAudit       = "Audit"
)

const (
	descRoles       lang.Text = "Manage roles and their capabilities."
	descAssignments lang.Text = "Assign roles to users."
	descAudit       lang.Text = "View the audit trail of actions."
)

// SVG Icon references.
const (
	IconRoles       = svg.Icon("access_roles")
	IconAssignments = svg.Icon("access_assignments")
	IconAudit       = svg.Icon("access_audit")
)
