// Package migrate creates the table access owns (the audit log). Run it at
// deploy time, next to auth's and rbac's own Migrate.
package migrate

import (
	"webtyp.com/access"
	"webtyp.com/ddl"
	"webtyp.com/model"
)

// Migrate creates access_audit if it does not exist.
//
// Example: err := migrate.Migrate(conn, compiler)
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	sorted, err := ddl.TopologicalSort([]model.Model{&access.AuditEntry{}})
	if err != nil {
		return err
	}
	return ddl.New(conn, ddlCompiler).Sync(sorted...)
}
