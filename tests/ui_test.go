package tests

import (
	"testing"

	"webtyp.com/access"
	"webtyp.com/access/seed"
	"webtyp.com/access/ui"
	"webtyp.com/model"

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

	caller := view.NewRecords().Caller(nil)
	screens := ui.Screens(caller)

	if len(screens) != 1 {
		t.Fatalf("expected 1 screen group, got %d", len(screens))
	}
}
