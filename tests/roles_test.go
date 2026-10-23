package tests

import (
	"testing"
	"webtyp.com/access"
	"webtyp.com/model"
)

func TestSaveRole_CreateThenUpdate(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)

	err := m.SaveRole("", access.RoleRecord{
		Code:        "recep",
		Name:        "Recepción",
		SessionTtl:  600,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = m.SaveRole("", access.RoleRecord{
		Code:        "recep",
		Name:        "Mesón",
	})
	if err != nil {
		t.Fatal(err)
	}

	roles, err := m.ListRoles()
	if err != nil {
		t.Fatal(err)
	}

	if len(roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(roles))
	}
	r := roles[0]
	if r.Name != "Mesón" {
		t.Errorf("expected role name Mesón, got %s", r.Name)
	}
	if r.SessionTtl != 0 {
		t.Errorf("expected session ttl 0, got %d", r.SessionTtl)
	}
}

func TestDeleteRole(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)

	m.SaveRole("", access.RoleRecord{Code: "recep", Name: "Recepción"})

	err := m.DeleteRole("", "recep")
	if err != nil {
		t.Fatal(err)
	}

	roles, _ := m.ListRoles()
	if len(roles) != 0 {
		t.Errorf("expected 0 roles, got %d", len(roles))
	}
}
