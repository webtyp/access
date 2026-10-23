package tests

import (
	"testing"
	"webtyp.com/model"
)

func TestEnsureRole_CreatesRoleAndUsers(t *testing.T) {
	m, authMod, rbacSvc := newEnv(t, model.AssuranceSingleFactor)

	err := m.EnsureRole("admin", "Admin", "", []string{"Ana@X.cl ", " ", "bob@x.cl"})
	if err != nil {
		t.Fatal(err)
	}

	role, err := rbacSvc.GetRoleByCode("p1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if role.Name != "Admin" {
		t.Errorf("expected role Name 'Admin', got '%s'", role.Name)
	}

	ana, err := authMod.UserByEmail("ana@x.cl")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := authMod.UserByEmail("bob@x.cl")
	if err != nil {
		t.Fatal(err)
	}

	usersInRole, err := rbacSvc.UsersInRole("p1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(usersInRole) != 2 {
		t.Fatalf("expected 2 users in role, got %d", len(usersInRole))
	}

	hasAna := false
	hasBob := false
	for _, id := range usersInRole {
		if id == ana.Id {
			hasAna = true
		}
		if id == bob.Id {
			hasBob = true
		}
	}
	if !hasAna || !hasBob {
		t.Errorf("expected ana and bob in role, got %v", usersInRole)
	}
}

func TestEnsureRole_Idempotent(t *testing.T) {
	m, authMod, rbacSvc := newEnv(t, model.AssuranceSingleFactor)

	err := m.EnsureRole("admin", "Admin", "", []string{"ana@x.cl", "bob@x.cl"})
	if err != nil {
		t.Fatal(err)
	}
	err = m.EnsureRole("admin", "Admin", "", []string{"ana@x.cl", "bob@x.cl"})
	if err != nil {
		t.Fatal(err)
	}

	roles, err := rbacSvc.ListRoles("p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(roles))
	}

	// users Ana, Bob (2 total users?) - auth module doesn't have a ListUsers, we assume they are two
	usersInRole, err := rbacSvc.UsersInRole("p1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(usersInRole) != 2 {
		t.Fatalf("expected 2 users in role, got %d", len(usersInRole))
	}

	ana1, _ := authMod.UserByEmail("ana@x.cl")
	ana2, _ := authMod.UserByEmail("ana@x.cl")
	if ana1.Id != ana2.Id {
		t.Fatalf("id mismatch for same email")
	}
}

func TestEnsureRole_ExistingUser(t *testing.T) {
	m, authMod, rbacSvc := newEnv(t, model.AssuranceSingleFactor)

	ana, err := authMod.CreateUser("ana@x.cl", "Ana", "")
	if err != nil {
		t.Fatal(err)
	}

	err = m.EnsureRole("admin", "Admin", "", []string{"ana@x.cl"})
	if err != nil {
		t.Fatal(err)
	}

	usersInRole, err := rbacSvc.UsersInRole("p1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(usersInRole) != 1 {
		t.Fatalf("expected 1 user in role, got %d", len(usersInRole))
	}
	if usersInRole[0] != ana.Id {
		t.Fatalf("expected existing user id %s in role, got %s", ana.Id, usersInRole[0])
	}
}
