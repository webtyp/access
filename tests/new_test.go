package tests

import (
	"testing"
	"webtyp.com/access"
	"webtyp.com/auth/authority"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/rbac"
)

func TestNew_RequiresEveryDep(t *testing.T) {
	db := &orm.DB{}
	ids := &seqIDs{}
	authMod := &authority.Module{}
	rbacSvc := &rbac.Service{}

	tests := []struct {
		name string
		deps access.Deps
		err  error
	}{
		{"no db", access.Deps{IDs: ids, Users: authMod, Roles: rbacSvc, ProjectID: "p1", Assurance: model.AssuranceSingleFactor}, access.ErrNoDB},
		{"no ids", access.Deps{DB: db, Users: authMod, Roles: rbacSvc, ProjectID: "p1", Assurance: model.AssuranceSingleFactor}, access.ErrNoIDs},
		{"no users", access.Deps{DB: db, IDs: ids, Roles: rbacSvc, ProjectID: "p1", Assurance: model.AssuranceSingleFactor}, access.ErrNoUsers},
		{"no roles", access.Deps{DB: db, IDs: ids, Users: authMod, ProjectID: "p1", Assurance: model.AssuranceSingleFactor}, access.ErrNoRoles},
		{"no project", access.Deps{DB: db, IDs: ids, Users: authMod, Roles: rbacSvc, Assurance: model.AssuranceSingleFactor}, access.ErrNoProject},
		{"zero assurance", access.Deps{DB: db, IDs: ids, Users: authMod, Roles: rbacSvc, ProjectID: "p1", Assurance: model.AssuranceNone}, access.ErrAssurance},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := access.New(tt.deps)
			if err != tt.err {
				t.Fatalf("expected %v, got %v", tt.err, err)
			}
		})
	}
}
