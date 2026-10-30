package tests

import (
	"errors"
	"testing"
	"webtyp.com/access"
	"webtyp.com/auth"
	"webtyp.com/model"
	"webtyp.com/rbac"
	"webtyp.com/router/loopback"
)

// failingUsers is a Users whose list fails: the store is down.
type failingUsers struct {
	access.Users
}

var errUsersDown = errors.New("users store down")

func (failingUsers) ListUsers() ([]auth.User, error) { return nil, errUsersDown }

func TestAssignUser_PersonWithoutEmail(t *testing.T) {
	m, authMod, _ := newEnv(t, model.AssuranceSingleFactor)
	m.SaveRole("seed", access.RoleRecord{Code: "reception", Name: "Receptionist"})
	staff, _ := authMod.CreateUser("", "Staff One", "")

	a, err := m.AssignUser("admin-1", staff.Id, "reception")
	if err != nil {
		t.Fatalf("AssignUser: %v", err)
	}

	expectedId := "reception/" + staff.Id
	if a.Id != expectedId || a.UserId != staff.Id || a.Name != "Staff One" || a.RoleCode != "reception" {
		t.Errorf("Unexpected assignment returned: %+v", a)
	}

	assignments, _ := m.ListAssignments()
	found := false
	for _, as := range assignments {
		if as.Id == expectedId {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Assignment not found in list")
	}

	audit, _ := m.ListAudit()
	if len(audit) == 0 {
		t.Fatalf("No audit entries found")
	}
	lastAudit := audit[0] // Newest first
	if lastAudit.Action != access.AuditAssign {
		t.Errorf("Expected audit Action 'role.assign', got %q", lastAudit.Action)
	}
	if lastAudit.Target != "reception" {
		t.Errorf("Expected audit Target 'reception', got %q", lastAudit.Target)
	}
	if lastAudit.Detail != "Staff One" {
		t.Errorf("Expected audit Detail 'Staff One', got %q", lastAudit.Detail)
	}
}

func TestAssignUser_Errors(t *testing.T) {
	m, authMod, _ := newEnv(t, model.AssuranceSingleFactor)
	m.SaveRole("seed", access.RoleRecord{Code: "reception", Name: "Receptionist"})
	staff, _ := authMod.CreateUser("", "Staff One", "")

	tests := []struct {
		userID   string
		roleCode string
		expected error
		name     string
	}{
		{"", "reception", access.ErrNoPerson, "no user id"},
		{staff.Id, "", access.ErrNoRoleCode, "no role code"},
		{"missing-id", "reception", access.ErrUserNotFound, "user not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := m.AssignUser("admin-1", tt.userID, tt.roleCode)
			if err != tt.expected {
				t.Errorf("AssignUser expected %v, got %v", tt.expected, err)
			}
		})
	}

	t.Run("role not found", func(t *testing.T) {
		_, err := m.AssignUser("admin-1", staff.Id, "nope")
		if !rbac.IsRoleNotFound(err) {
			t.Errorf("expected role not found, got %v", err)
		}
	})
}

func TestAssignByEmail_Invite(t *testing.T) {
	m, authMod, _ := newEnv(t, model.AssuranceSingleFactor)
	m.SaveRole("seed", access.RoleRecord{Code: "reception", Name: "Receptionist"})

	_, err := m.AssignByEmail("admin-1", "new@example.com", "reception")
	if err != nil {
		t.Fatalf("AssignByEmail: %v", err)
	}

	u, err := authMod.UserByEmail("new@example.com")
	if err != nil {
		t.Fatalf("User was not created: %v", err)
	}

	assignments, _ := m.ListAssignments()
	found := false
	for _, as := range assignments {
		if as.UserId == u.Id && as.RoleCode == "reception" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Role was not given to the created user")
	}
}

func TestListPeople_WithRoles(t *testing.T) {
	m, authMod, _ := newEnv(t, model.AssuranceSingleFactor)
	m.SaveRole("seed", access.RoleRecord{Code: "reception", Name: "Receptionist"})
	staff, _ := authMod.CreateUser("", "Staff One", "")
	ana, _ := authMod.CreateUser("ana@example.com", "Ana", "")

	_, err := m.AssignUser("admin-1", staff.Id, "reception")
	if err != nil {
		t.Fatalf("AssignUser: %v", err)
	}

	people, err := m.ListPeople()
	if err != nil {
		t.Fatalf("ListPeople: %v", err)
	}

	if len(people) != 2 {
		t.Fatalf("Expected 2 people, got %d", len(people))
	}

	staffFound, anaFound := false, false
	for _, p := range people {
		if p.Id == staff.Id {
			staffFound = true
			if p.Roles != "Receptionist" || p.Email != "" {
				t.Errorf("Staff: expected Roles='Receptionist', Email='', got Roles=%q, Email=%q", p.Roles, p.Email)
			}
		} else if p.Id == ana.Id {
			anaFound = true
			if p.Roles != "" || p.Email != "ana@example.com" {
				t.Errorf("Ana: expected Roles='', Email='ana@example.com', got Roles=%q, Email=%q", p.Roles, p.Email)
			}
		}
	}

	if !staffFound || !anaFound {
		t.Errorf("Did not find both staff and ana in people list")
	}
}

func TestListPeople_UsersDown(t *testing.T) {
	db, ids, _, authMod, rbacSvc := newEnvDetails(t)

	// Create a new module with failingUsers
	mDown, err := access.New(access.Deps{
		DB:        db,
		IDs:       ids,
		Users:     failingUsers{Users: authMod},
		Roles:     rbacSvc,
		ProjectID: "p1",
		Assurance: model.AssuranceSingleFactor,
	})
	if err != nil {
		t.Fatalf("New access.Module: %v", err)
	}

	_, err = mDown.ListPeople()
	if err != errUsersDown {
		t.Errorf("Expected errUsersDown, got %v", err)
	}

	caller := loopback.ActingAs("p1", "admin-1", mDown)
	var res access.PersonList
	caller.Call("access."+access.OpListPeople, nil, &res, func(e error) {
		err = e
	})
	if err == nil {
		t.Errorf("Expected error from loopback call")
	}
}

func TestEnsureRoleFor(t *testing.T) {
	m, authMod, _ := newEnv(t, model.AssuranceSingleFactor)
	staff, _ := authMod.CreateUser("", "Staff One", "")

	err := m.EnsureRoleFor("admin", "Administrator", "", []string{staff.Id})
	if err != nil {
		t.Fatalf("EnsureRoleFor: %v", err)
	}

	err = m.EnsureRoleFor("admin", "Administrator", "", []string{staff.Id})
	if err != nil {
		t.Fatalf("EnsureRoleFor twice: %v", err)
	}

	roles, _ := m.ListRoles()
	count := 0
	for _, r := range roles {
		if r.Code == "admin" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Expected role 'admin' to exist exactly once, found %d times", count)
	}

	assignments, _ := m.ListAssignments()
	found := false
	for _, as := range assignments {
		if as.UserId == staff.Id && as.RoleCode == "admin" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Staff does not hold the admin role")
	}
}

func TestEnsureRoleFor_Errors(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)

	err := m.EnsureRoleFor("admin", "Administrator", "", []string{"missing-id"})
	if err != access.ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}

	err = m.EnsureRoleFor("admin", "Administrator", "", []string{""})
	if err != access.ErrNoPerson {
		t.Errorf("Expected ErrNoPerson, got %v", err)
	}

	err = m.EnsureRoleFor("", "Administrator", "", []string{"some-id"})
	if err != access.ErrNoRoleCode {
		t.Errorf("Expected ErrNoRoleCode, got %v", err)
	}
}
