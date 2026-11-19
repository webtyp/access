package tests

import (
	"testing"

	"webtyp.com/access"
	"webtyp.com/access/migrate"
	"webtyp.com/auth"
	"webtyp.com/auth/authority"
	"webtyp.com/ddl"
	"webtyp.com/events/inprocess"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/rbac"
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
	authMod, err := authority.New(db, auth.Config{IDs: ids, Events: &inprocess.Broker{}})
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
	authMod, err := authority.New(db, auth.Config{IDs: ids, Events: &inprocess.Broker{}})
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
