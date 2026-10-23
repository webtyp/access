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

	caller.Call("access."+access.OpAssignRoles, &access.AssignRolesArgs{
		Records: []access.Assignment{{Email: "luz@x.cl", RoleCode: "recep"}},
	}, nil, func(e error) { err = e })
	if err != nil {
		t.Fatal(err)
	}

	var assignments access.AssignmentList
	caller.Call("access."+access.OpListAssignments, nil, &assignments, func(e error) { err = e })
	if err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 1 {
		t.Errorf("expected 1 assignment, got %d", len(assignments))
	}

	u, _ := authMod.UserByEmail("luz@x.cl")
	caller.Call("access."+access.OpRevokeAssignments, &access.IDsArgs{
		Ids: []string{"recep/" + u.Id},
	}, nil, func(e error) { err = e })
	if err != nil {
		t.Fatal(err)
	}

	var audit access.AuditEntryList
	caller.Call("access."+access.OpListAudit, nil, &audit, func(e error) { err = e })
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) != 3 {
		t.Errorf("expected 3 audit entries, got %d", len(audit))
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

	if len(regSingle.routes) != 7 {
		t.Errorf("expected 7 operations mounted, got %d", len(regSingle.routes))
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
