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

	if len(screens) != 3 {
		t.Fatalf("expected 3 screens, got %d", len(screens))
	}

	expectedPerms := []model.Resource{access.ResourceRole, access.ResourceAssignment, access.ResourceAudit}
	for i, s := range screens {
		if err := s.Validate(); err != nil {
			t.Errorf("screen %d validate: %v", i, err)
		}
		if s.Permission != expectedPerms[i] {
			t.Errorf("screen %d permission expected %q, got %q", i, expectedPerms[i], s.Permission)
		}
	}

	// Verify AssignmentModel's role_code has Options pointing to RoleRecordModel
	var roleCodeField *model.Field
	for i, f := range access.AssignmentModel.Fields {
		if f.Name == "role_code" {
			roleCodeField = &access.AssignmentModel.Fields[i]
			break
		}
	}
	if roleCodeField == nil {
		t.Fatalf("AssignmentModel missing role_code field")
	}
	if roleCodeField.Options != &access.RoleRecordModel {
		t.Errorf("AssignmentModel role_code Options not set to RoleRecordModel")
	}
}
