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

func TestScreens(t *testing.T) {
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

	group := screens[0]
	if group.ID != ui.ID {
		t.Errorf("expected group ID %q, got %q", ui.ID, group.ID)
	}
	if group.Help == "" {
		t.Errorf("expected non-empty help for group")
	}
	if group.Permission != "" {
		t.Errorf("expected empty permission for group, got %q", group.Permission)
	}

	if len(group.Children) != 4 {
		t.Fatalf("expected 4 children screens, got %d", len(group.Children))
	}

	expectedPerms := []model.Resource{access.ResourceAssignment, access.ResourceRole, access.ResourceAssignment, access.ResourceAudit}
	for i, s := range group.Children {
		if err := s.Validate(); err != nil {
			t.Errorf("screen %d validate: %v", i, err)
		}
		if s.Permission != expectedPerms[i] {
			t.Errorf("screen %d permission expected %q, got %q", i, expectedPerms[i], s.Permission)
		}
	}

	// Verify AssignmentModel's role_code has Options pointing to RoleRecordModel
	var roleCodeField *model.Field
	var userIdField *model.Field
	for i, f := range access.AssignmentModel.Fields {
		if f.Name == "role_code" {
			roleCodeField = &access.AssignmentModel.Fields[i]
		}
		if f.Name == "user_id" {
			userIdField = &access.AssignmentModel.Fields[i]
		}
	}
	if roleCodeField == nil {
		t.Fatalf("AssignmentModel missing role_code field")
	}
	if roleCodeField.Options != &access.RoleRecordModel {
		t.Errorf("AssignmentModel role_code Options not set to RoleRecordModel")
	}
	if userIdField != nil && userIdField.Options != &access.PersonModel {
		t.Errorf("AssignmentModel user_id Options not set to PersonModel")
	}
}
