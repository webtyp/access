package ui

import "webtyp.com/svg"

// ID is this module's root identity on client nav rails.
const ID = "access"

// Labels for navigation rail and headers.
const (
	LabelRoles       = "Roles"
	LabelAssignments = "Asignaciones"
	LabelAudit       = "Auditoría"
)

// SVG Icon references.
const (
	IconRoles       = svg.Icon("access_roles")
	IconAssignments = svg.Icon("access_assignments")
	IconAudit       = svg.Icon("access_audit")
)
