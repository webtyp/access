package tests

import (
	"testing"
	"webtyp.com/access"
	"webtyp.com/model"
	"webtyp.com/rbac"
)

func TestAssign_CreatesUserAndLists(t *testing.T) {
	m, authMod, _ := newEnv(t, model.AssuranceSingleFactor)
	m.SaveRole("", access.RoleRecord{Code: "recep", Name: "Recepción"})

	a, err := m.Assign("", "luz@x.cl", "recep")
	if err != nil {
		t.Fatal(err)
	}

	u, _ := authMod.UserByEmail("luz@x.cl")
	expectedId := "recep/" + u.Id
	if a.Id != expectedId {
		t.Errorf("expected id %s, got %s", expectedId, a.Id)
	}

	assignments, err := m.ListAssignments()
	if err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(assignments))
	}
	if assignments[0].Email != "luz@x.cl" {
		t.Errorf("expected email luz@x.cl, got %s", assignments[0].Email)
	}
}

func TestAssign_UnknownRole(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)

	_, err := m.Assign("", "luz@x.cl", "nope")
	if !rbac.IsRoleNotFound(err) {
		t.Errorf("expected IsRoleNotFound, got %v", err)
	}
}

func TestRevoke(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)
	m.SaveRole("", access.RoleRecord{Code: "recep", Name: "Recepción"})
	a, _ := m.Assign("", "luz@x.cl", "recep")

	err := m.Revoke("", a.Id)
	if err != nil {
		t.Fatal(err)
	}

	assignments, _ := m.ListAssignments()
	if len(assignments) != 0 {
		t.Errorf("expected 0 assignments, got %d", len(assignments))
	}
}

func TestRevoke_BadID(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)
	err := m.Revoke("", "only-one-part")
	if err != access.ErrAssignmentID {
		t.Errorf("expected ErrAssignmentID, got %v", err)
	}
}
