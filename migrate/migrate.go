// Package migrate creates the tables the entry module owns: users (authority), roles (rbac), and audit log.
package migrate

import (
	"webtyp.com/access"
	"webtyp.com/auth/authority"
	"webtyp.com/ddl"
	"webtyp.com/model"
	"webtyp.com/rbac"
)

// Migrate creates the tables the entry module owns: users, roles, and audit log.
//
// Example: err := migrate.Migrate(conn, compiler)
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	if err := authority.Migrate(conn, ddlCompiler); err != nil {
		return err
	}
	if err := rbac.Migrate(conn, ddlCompiler); err != nil {
		return err
	}
	sorted, err := ddl.TopologicalSort([]model.Model{&access.AuditEntry{}})
	if err != nil {
		return err
	}
	return ddl.New(conn, ddlCompiler).Sync(sorted...)
}
