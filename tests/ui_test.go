package tests

import (
	"testing"
	"webtyp.com/access"
	"webtyp.com/access/seed"
	"webtyp.com/access/ui"
	"webtyp.com/model"
	"webtyp.com/router/loopback"
	"webtyp.com/view"
)

func TestItemizers(t *testing.T) {
	r := &access.RoleRecord{Id: "1", Code: "admin", Name: "Admin"}
	it := r.Item()
	if it.ID != "1" || it.Label != "Admin" || it.Description != "admin" {
		t.Errorf("unexpected RoleRecord.Item(): %+v", it)
	}

	a := &access.Assignment{Id: "admin/u1", Email: "admin@test.com", RoleCode: "admin", Name: "Boss"}
	itA := a.Item()
	if itA.ID != "admin/u1" || itA.Label != "Boss" || itA.Description != "admin" {
		t.Errorf("unexpected Assignment.Item(): %+v", itA)
	}

	p := &access.Person{Id: "u1", Name: "Ana", Email: "ana@example.com", Roles: "Receptionist"}
	itP := p.Item()
	if itP.ID != "u1" || itP.Label != "Ana" || itP.Description != "ana@example.com · Receptionist" {
		t.Errorf("unexpected Person.Item(): %+v", itP)
	}

	e := &access.AuditEntry{Id: "aud1", ActorEmail: "admin@test.com", Action: "role.save", Target: "admin"}
	itE := e.Item()
	if itE.ID != "aud1" || itE.Label != "admin@test.com" || itE.Description != "role.save · admin" {
		t.Errorf("unexpected AuditEntry.Item(): %+v", itE)
	}
}

func TestUIScreens(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)
	if err := seed.Load(m); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}

	rawCaller := loopback.ActingAs("p1", "admin-1", m)
	caller := view.NewRecords().Caller(rawCaller)
	screens := ui.Screens(caller)

	if len(screens) != 1 {
		t.Fatalf("expected 1 screen group, got %d", len(screens))
	}
}

func TestPresentersReload(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)
	if err := seed.Load(m); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}

	rawCaller := loopback.ActingAs("p1", "admin-1", m)
	caller := view.NewRecords().Caller(rawCaller)

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
