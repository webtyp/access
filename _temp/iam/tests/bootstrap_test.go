//go:build !wasm

package tests

import (
	"testing"

	"webtyp.com/model"
	"github.com/veltylabs/iam/config"
)

func TestEnsureRoleIdempotentAndAssignsEmails(t *testing.T) {
	_, authMod, rbacSvc, _ := setup(t)

	if err := rbacSvc.CreatePermission(testProjectID, permExampleRead, "Read example", exampleResource, model.Read); err != nil {
		t.Fatalf("CreatePermission: %v", err)
	}

	email := "op1@velty.cl"
	if err := config.EnsureRole(rbacSvc, authMod, testProjectID, roleExampleID, roleExampleCode, "Example Admin", "", []string{email}); err != nil {
		t.Fatalf("EnsureRole (1st): %v", err)
	}
	if err := rbacSvc.AssignPermission(testProjectID, roleExampleID, permExampleRead); err != nil {
		t.Fatalf("AssignPermission: %v", err)
	}
	// Segunda llamada: idempotente, no debe fallar ni duplicar.
	if err := config.EnsureRole(rbacSvc, authMod, testProjectID, roleExampleID, roleExampleCode, "Example Admin", "", []string{email}); err != nil {
		t.Fatalf("EnsureRole (2nd): %v", err)
	}

	u, err := authMod.UserByEmail(email)
	if err != nil {
		t.Fatalf("UserByEmail(%s): %v", email, err)
	}
	if !rbacSvc.Can(testProjectID, u.Id, exampleResource, model.Read) {
		t.Errorf("expected %s to have %s:Read via %s", email, permExampleRead, roleExampleID)
	}
}

func TestEnsureRoleRejectsNilDeps(t *testing.T) {
	if err := config.EnsureRole(nil, nil, testProjectID, roleExampleID, roleExampleCode, "Example Admin", "", nil); err == nil {
		t.Errorf("expected error when rbac service and authority module are nil")
	}
}
