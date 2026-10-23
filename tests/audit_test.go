package tests

import (
	"testing"
	"webtyp.com/access"
	"webtyp.com/model"
)

func TestAudit_RecordsActorEmail(t *testing.T) {
	m, authMod, _ := newEnv(t, model.AssuranceSingleFactor)
	adm, _ := authMod.CreateUser("adm@x.cl", "Admin", "")

	m.SaveRole(adm.Id, access.RoleRecord{Code: "recep", Name: "Recepción"})
	m.Assign(adm.Id, "luz@x.cl", "recep")

	audit, err := m.ListAudit()
	if err != nil {
		t.Fatal(err)
	}

	if len(audit) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(audit))
	}

	if audit[0].Action != access.AuditAssign {
		t.Errorf("expected first action role.assign, got %s", audit[0].Action)
	}
	if audit[1].Action != access.AuditRoleSave {
		t.Errorf("expected second action role.save, got %s", audit[1].Action)
	}
	if audit[0].ActorEmail != "adm@x.cl" || audit[1].ActorEmail != "adm@x.cl" {
		t.Errorf("expected ActorEmail adm@x.cl")
	}
}
