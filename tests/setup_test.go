package tests

import (
	"testing"
	"webtyp.com/access"
	"webtyp.com/access/migrate"
	"webtyp.com/auth"
	"webtyp.com/auth/authority"
	"webtyp.com/ddl"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/rbac"
	"webtyp.com/router"
	"webtyp.com/storage/mem"
)

func newEnvDetails(t *testing.T) (*orm.DB, model.IDGenerator, *access.Module, *authority.Module, *rbac.Service) {
	t.Helper()
	db := orm.New(mem.New())
	var c ddl.Compiler
	if ddlCompiler, ok := db.RawConn().(ddl.Compiler); ok {
		c = ddlCompiler
	}
	if c != nil {
		if err := authority.Migrate(db.RawConn(), c); err != nil {
			t.Fatal(err)
		}
		if err := rbac.Migrate(db.RawConn(), c); err != nil {
			t.Fatal(err)
		}
		if err := migrate.Migrate(db.RawConn(), c); err != nil {
			t.Fatal(err)
		}
	}

	ids := &seqIDs{}
	authMod, err := authority.New(db, auth.Config{IDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	rbacSvc, err := rbac.New(db)
	if err != nil {
		t.Fatal(err)
	}

	m, err := access.New(access.Deps{
		DB:        db,
		IDs:       ids,
		Users:     authMod,
		Roles:     rbacSvc,
		ProjectID: "p1",
		Assurance: model.AssuranceSingleFactor,
	})
	if err != nil {
		t.Fatal(err)
	}
	return db, ids, m, authMod, rbacSvc
}

func newEnv(t *testing.T, assurance model.Assurance) (*access.Module, *authority.Module, *rbac.Service) {
	t.Helper()
	db := orm.New(mem.New())
	var c ddl.Compiler
	if ddlCompiler, ok := db.RawConn().(ddl.Compiler); ok {
		c = ddlCompiler
	}
	if c != nil {
		if err := authority.Migrate(db.RawConn(), c); err != nil {
			t.Fatal(err)
		}
		if err := rbac.Migrate(db.RawConn(), c); err != nil {
			t.Fatal(err)
		}
		if err := migrate.Migrate(db.RawConn(), c); err != nil {
			t.Fatal(err)
		}
	}

	ids := &seqIDs{}
	authMod, err := authority.New(db, auth.Config{IDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	rbacSvc, err := rbac.New(db)
	if err != nil {
		t.Fatal(err)
	}

	m, err := access.New(access.Deps{
		DB:        db,
		IDs:       ids,
		Users:     authMod,
		Roles:     rbacSvc,
		ProjectID: "p1",
		Assurance: assurance,
	})
	if err != nil {
		t.Fatal(err)
	}
	return m, authMod, rbacSvc
}

type seqIDs struct {
	next int
}

func (s *seqIDs) NewID() string {
	s.next++
	return "id-" + string(rune(s.next+'0')) // simplified for tests
}

type mockOpRegistry struct {
	routes map[string]*mockRoute
}

func (r *mockOpRegistry) Operation(name string, h router.HandlerFunc) router.Route {
	if r.routes == nil {
		r.routes = make(map[string]*mockRoute)
	}
	route := &mockRoute{name: name}
	r.routes[name] = route
	return route
}

type mockRoute struct {
	name        string
	resource    model.Resource
	action      model.Action
	requiresMFA bool
}

func (r *mockRoute) Requires(res model.Resource, act model.Action) router.Route {
	r.resource = res
	r.action = act
	return r
}

func (r *mockRoute) RequiresMFA() router.Route {
	r.requiresMFA = true
	return r
}

func (r *mockRoute) Accepts(model.Fielder) router.Route { return r }
func (r *mockRoute) Public() router.Route { return r }
func (r *mockRoute) Authenticated() router.Route { return r }
func (r *mockRoute) Describe(string) router.Route { return r }
