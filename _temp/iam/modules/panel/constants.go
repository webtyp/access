package panel

import (
	"webtyp.com/svg"
)

const (
	ModuleProjects = "projects"
	ModuleRoles    = "roles"
	ModuleUsers    = "users"
	ModuleAudit    = "audit"
)

// Panel aloja los assets e iconos del módulo panel de administración.
type Panel struct{}


const (
	IconProjects = svg.Icon("iam-projects")
	IconRoles    = svg.Icon("iam-roles")
	IconUsers    = svg.Icon("iam-users")
	IconAudit    = svg.Icon("iam-audit")
)

const (
	IDProjectsList   = "iam-projects-list"
	IDProjectsNew    = "iam-projects-new"
	IDRolesProject   = "iam-roles-project"
	IDRolesList      = "iam-roles-list"
	IDRolesNew       = "iam-roles-new"
	IDUsersForm      = "iam-users-form"
	IDUsersList      = "iam-users-list"
	IDAuditList      = "iam-audit-list"
)
