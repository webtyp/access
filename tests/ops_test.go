package tests

import (
	"testing"
	"webtyp.com/access"
	"webtyp.com/model"
	"webtyp.com/router/loopback"
)

func TestOps_ThroughCaller(t *testing.T) {
	m, authMod, _ := newEnv(t, model.AssuranceSingleFactor)
	adm, _ := authMod.CreateUser("adm@x.cl", "Admin", "")

	caller := loopback.ActingAs("p1", adm.Id, m)

	var err error
	caller.Call("access."+access.OpSaveRoles, &access.SaveRolesArgs{
		Records: []access.RoleRecord{{Code: "recep", Name: "Recep", SessionTtl: 600}},
	}, nil, func(e error) { err = e })
	if err != nil {
		t.Fatal(err)
	}

}

func TestOps_AssignRecords(t *testing.T) {
	m, authMod, _ := newEnv(t, model.AssuranceSingleFactor)
	m.SaveRole("seed", access.RoleRecord{Code: "reception", Name: "Receptionist"})
	adm, _ := authMod.CreateUser("adm@x.cl", "Admin", "")
	staff, _ := authMod.CreateUser("", "Staff One", "")

	caller := loopback.ActingAs("p1", adm.Id, m)

	tests := []struct {
		records  []access.Assignment
		expectEr bool
		name     string
	}{
		{[]access.Assignment{{UserId: staff.Id, RoleCode: "reception"}}, false, "userId set"},
		{[]access.Assignment{{Email: "invited@example.com", RoleCode: "reception"}}, false, "email set"},
		{[]access.Assignment{{RoleCode: "reception"}}, true, "neither set"},
		{[]access.Assignment{{UserId: staff.Id, Email: "other@example.com", RoleCode: "reception"}}, false, "both set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			caller.Call("access."+access.OpAssignRoles, &access.AssignRolesArgs{
				Records: tt.records,
			}, nil, func(e error) { err = e })
			if tt.expectEr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectEr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}

	_, err := authMod.UserByEmail("other@example.com")
	if err == nil {
		t.Errorf("expected other@example.com to not exist")
	}
}

func TestOps_EmptyBatch(t *testing.T) {
	m, authMod, _ := newEnv(t, model.AssuranceSingleFactor)
	adm, _ := authMod.CreateUser("adm@x.cl", "Admin", "")

	caller := loopback.ActingAs("p1", adm.Id, m)

	var err error
	caller.Call("access."+access.OpSaveRoles, &access.SaveRolesArgs{}, nil, func(e error) { err = e })
	if err == nil {
		t.Fatal("expected error")
	}
	// status 400 check is inside loopback error but we just check error is returned
}

func TestOps_Gates(t *testing.T) {
	mSingle, _, _ := newEnv(t, model.AssuranceSingleFactor)
	regSingle := &mockOpRegistry{}
	mSingle.MountOperations(regSingle)

	if len(regSingle.routes) != 8 {
		t.Errorf("expected 8 operations mounted, got %d", len(regSingle.routes))
	}

	listPeopleRoute, ok := regSingle.routes[access.OpListPeople]
	if !ok {
		t.Errorf("expected OpListPeople to be mounted")
	} else {
		if listPeopleRoute.resource != access.ResourceAssignment || listPeopleRoute.action != model.Read {
			t.Errorf("expected OpListPeople to require ResourceAssignment Read")
		}
	}

	for _, r := range regSingle.routes {
		if r.requiresMFA {
			t.Errorf("did not expect RequiresMFA on single factor")
		}
	}

	mMulti, _, _ := newEnv(t, model.AssuranceMultiFactor)
	regMulti := &mockOpRegistry{}
	mMulti.MountOperations(regMulti)

	for _, r := range regMulti.routes {
		if !r.requiresMFA {
			t.Errorf("expected RequiresMFA on multi factor for %s", r.name)
		}
	}
}
