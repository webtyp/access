package tests

import (
	"testing"
	"webtyp.com/access"
	"webtyp.com/access/seed"
	"webtyp.com/access/ui"
	"webtyp.com/model"
	"webtyp.com/router/loopback"
)

func TestRowsDeclaredOnFields(t *testing.T) {
	r := &access.RoleRecord{Id: "1", Code: "admin", Name: "Admin"}
	rowR := model.RowOf(r)
	if rowR.ID != "1" || rowR.Title != "Admin" || rowR.Detail != "admin" {
		t.Errorf("unexpected RoleRecord row: %+v", rowR)
	}

	a := &access.Assignment{Id: "admin/u1", Email: "admin@test.com", RoleCode: "admin", Name: "Boss"}
	rowA := model.RowOf(a)
	if rowA.ID != "admin/u1" || rowA.Title != "admin@test.com" || rowA.Detail != "admin · Boss" {
		t.Errorf("unexpected Assignment row: %+v", rowA)
	}

	e := &access.AuditEntry{Id: "aud1", ActorEmail: "admin@test.com", Action: "role.save", Target: "admin"}
	rowE := model.RowOf(e)
	if rowE.ID != "aud1" || rowE.Title != "admin@test.com" || rowE.Detail != "role.save · admin" {
		t.Errorf("unexpected AuditEntry row: %+v", rowE)
	}
}

func TestUIBrowserModules(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)
	if err := seed.Load(m); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}

	caller := loopback.ActingAs("p1", "admin-1", m)
	mods, err := ui.Browser(caller, &seqIDs{})
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}

	if len(mods) != 3 {
		t.Fatalf("expected 3 modules (Roles, Assignments, Audit), got %d", len(mods))
	}

	expectedIDs := []string{"access.roles", "access.assignments", "access.audit"}
	for i, exp := range expectedIDs {
		if mods[i].ModelName() != exp {
			t.Errorf("module %d ID: expected %q, got %q", i, exp, mods[i].ModelName())
		}
		if mods[i].View() == nil {
			t.Errorf("module %d view is nil", i)
		}
	}
}

func TestPresentersReload(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)
	if err := seed.Load(m); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}

	caller := loopback.ActingAs("p1", "admin-1", m)
	caller = view.NewRecords().Caller(caller)

	rolesPres := access.NewRolesView(caller)
	rolesPres.Reload(func(err error) {
		if err != nil {
			t.Fatalf("roles reload: %v", err)
		}
		items := rolesPres.Items()
		if len(items) != 3 {
			t.Errorf("expected 3 seeded roles, got %d", len(items))
		}
	})

	assignPres := access.NewAssignmentsView(caller)
	assignPres.Reload(func(err error) {
		if err != nil {
			t.Fatalf("assignments reload: %v", err)
		}
		items := assignPres.Items()
		if len(items) != 1 {
			t.Errorf("expected 1 seeded assignment, got %d", len(items))
		}
	})

	auditPres := access.NewAuditView(caller)
	auditPres.Reload(func(err error) {
		if err != nil {
			t.Fatalf("audit reload: %v", err)
		}
		items := auditPres.Items()
		if len(items) == 0 {
			t.Errorf("expected audit entries from seed, got 0")
		}
	})
}
