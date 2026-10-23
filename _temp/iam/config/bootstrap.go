package config

import (
	"webtyp.com/auth/authority"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/rbac"
)

// EnsureRole crea el rol (si no existe) y lo asigna de forma idempotente a
// cada email de la lista, creando el usuario en auth si todavía no existe.
// No crea ni asigna permisos: adjuntar permisos al rol es responsabilidad
// de quien llama — cada app declara su propia política (ver
// ARCHITECTURE.md §1: "Policy belongs to the consumer" de tinywasm/rbac).
func EnsureRole(rbacSvc *rbac.Service, authMod *authority.Module, projectID, roleID string, roleCode model.RoleCode, roleName, roleDescription string, emails []string) error {
	if rbacSvc == nil {
		return fmt.Errf("bootstrap: rbac service is nil")
	}
	if authMod == nil {
		return fmt.Errf("bootstrap: authority module is nil")
	}
	if err := rbacSvc.CreateRole(projectID, roleID, roleCode, roleName, roleDescription); err != nil && !isAlreadyExistsErr(err) {
		return fmt.Errf("bootstrap: error creando rol %s: %v", roleID, err)
	}
	for i := 0; i < len(emails); i++ {
		email := fmt.TrimSpace(emails[i])
		if email == "" {
			continue
		}
		u, err := authMod.UserByEmail(email)
		if err != nil {
			u, err = authMod.CreateUser(email, email, "")
			if err != nil {
				return fmt.Errf("bootstrap: error creando usuario para %s: %v", email, err)
			}
		}
		if err := rbacSvc.AssignRole(projectID, u.Id, roleID); err != nil && !isAlreadyExistsErr(err) {
			return fmt.Errf("bootstrap: error asignando rol a %s: %v", email, err)
		}
	}
	return nil
}

func isAlreadyExistsErr(err error) bool {
	if err == nil {
		return true
	}
	msg := err.Error()
	return fmt.Contains(msg, "already") || fmt.Contains(msg, "exists") || fmt.Contains(msg, "UNIQUE") || fmt.Contains(msg, "duplicate")
}
