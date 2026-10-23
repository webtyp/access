//go:build !wasm

package tests

import (
	"strings"
	"testing"

	"webtyp.com/base64"
	"webtyp.com/crypto/asym"
	"webtyp.com/json"
	"webtyp.com/router/mock"

	"github.com/veltylabs/iam/config"
)

func TestAudit_RecordsEveryMutation(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	adminU, err := b.Auth.UserByEmail("admin@example.com")
	if err != nil {
		adminU, _ = b.Auth.CreateUser("admin@example.com", "Admin", "")
	}

	// 1. Create Project
	hCreateProj := config.CreateProjectHandler(b.DB, b.IDs, b.Auth, b.RBAC)
	ctx1 := &mock.Context{InMethod: "POST", InPath: config.PathAdminProjects}
	ctx1.SetUserID(adminU.Id)
	ctx1.InBody = encodeBody(t, &config.AdminCreateProjectRequest{Name: "Audit Proj"})
	hCreateProj(ctx1)

	var projResp config.AdminProjectIDRequest
	_ = json.Decode(ctx1.ResponseBody(), &projResp)
	pID := projResp.ProjectId

	// 2. Set Project Key
	pub, _, err := asym.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	hKey := config.SetProjectKeyHandler(b.DB, b.IDs, b.Auth)
	ctx2 := &mock.Context{InMethod: "POST", InPath: config.PathAdminProjectKey}
	ctx2.SetUserID(adminU.Id)
	ctx2.InBody = encodeBody(t, &config.AdminProjectKeyRequest{ProjectId: pID, PublicKey: base64.URLEncode(pub)})
	hKey(ctx2)

	// 3. Create Role
	hCreateRole := config.CreateRoleHandler(b.DB, b.RBAC, b.IDs, b.Auth)
	ctx3 := &mock.Context{InMethod: "POST", InPath: config.PathAdminRoles}
	ctx3.SetUserID(adminU.Id)
	ctx3.InBody = encodeBody(t, &config.AdminCreateRoleRequest{ProjectId: pID, Code: "auditor", Name: "Auditor", Description: ""})
	hCreateRole(ctx3)

	// Fetch audit entries
	hAudit := config.ListAuditHandler(b.DB)
	ctxAudit := &mock.Context{InMethod: "GET", InPath: config.PathAdminAudit}
	ctxAudit.SetUserID(adminU.Id)
	hAudit(ctxAudit)

	var auditResp config.AdminAuditResponse
	_ = json.Decode(ctxAudit.ResponseBody(), &auditResp)

	if len(auditResp.Entries) < 2 {
		t.Fatalf("expected at least 2 audit entries, got %d", len(auditResp.Entries))
	}

	for _, entry := range auditResp.Entries {
		if entry.ActorEmail != "admin@example.com" {
			t.Errorf("expected actor email admin@example.com, got %s", entry.ActorEmail)
		}
	}
}

func TestAudit_ReadOnlyEndpoint(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	adminU, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")

	_ = config.RecordAudit(b.DB, b.IDs, "admin@example.com", config.AuditProjectCreate, "proj1", "detail")

	hAudit := config.ListAuditHandler(b.DB)
	ctxAudit := &mock.Context{InMethod: "GET", InPath: config.PathAdminAudit}
	ctxAudit.SetUserID(adminU.Id)
	hAudit(ctxAudit)

	if ctxAudit.Status != 200 {
		t.Fatalf("ListAudit status %d, want 200", ctxAudit.Status)
	}

	var auditResp config.AdminAuditResponse
	_ = json.Decode(ctxAudit.ResponseBody(), &auditResp)
	if len(auditResp.Entries) != 1 || auditResp.Entries[0].Action != config.AuditProjectCreate {
		t.Fatalf("unexpected audit list: %+v", auditResp.Entries)
	}
}

// TestAudit_RecordsOriginDenial: mutación desde un origen ajeno → 403 + fila
// panel.origin_denied.
func TestAudit_RecordsOriginDenial(t *testing.T) {
	b, r := setupPanel(t, "admin@example.com")

	ctx := crossSiteCtx(t, b, "admin@example.com")
	ctx.InMethod = "POST"
	ctx.InPath = config.PathAdminProjects
	ctx.InBody = encodeBody(t, &config.AdminCreateProjectRequest{Name: "Denied App"})
	r.Invoke("POST", config.PathAdminProjects, ctx)
	if ctx.Status != 403 {
		t.Fatalf("status %d, want 403", ctx.Status)
	}

	entry := lastAudit(t, b.DB)
	if entry.Action != config.AuditOriginDenied {
		t.Errorf("action: got %q, want %q", entry.Action, config.AuditOriginDenied)
	}
	if entry.Target != "admin@example.com" {
		t.Errorf("target: got %q, want the resolved admin email", entry.Target)
	}
}

// TestAudit_MachineMutations: AssignRole y ResolveUser auditan sus acciones
func TestAudit_MachineMutations(t *testing.T) {
	backend := setupBackend(t)
	const projectID = "proj-audit-machine"
	_ = config.CreateProject(backend.DB, projectID, "Audit Machine", "")

	// 1. ResolveUser creating a new user
	hResolve := config.ResolveUser(backend.DB, backend.Auth, backend.IDs)
	ctxRes := &mock.Context{InMethod: "POST", InPath: "/api/users/resolve"}
	ctxRes.SetUserID(projectID)
	var body []byte
	_ = json.Encode(&config.ResolveUserRequest{Email: "audituser@example.com", Name: "Audit User"}, &body)
	ctxRes.InBody = body
	hResolve(ctxRes)
	if ctxRes.Status != 200 {
		t.Fatalf("ResolveUser status %d, want 200", ctxRes.Status)
	}

	entry1 := lastAudit(t, backend.DB)
	if entry1.Action != config.AuditIdentityCreatedByClient {
		t.Errorf("action: got %q, want %q", entry1.Action, config.AuditIdentityCreatedByClient)
	}
	if entry1.Target != "audituser@example.com" {
		t.Errorf("target: got %q, want %q", entry1.Target, "audituser@example.com")
	}

	// 2. AssignRole
	_ = backend.RBAC.CreateRole(projectID, "r1", "client-role", "Client Role", "")
	hAssign := config.AssignRole(backend.DB, backend.RBAC, backend.IDs)
	ctxAssign := &mock.Context{InMethod: "POST", InPath: "/api/roles/assign"}
	ctxAssign.SetUserID(projectID)
	var body2 []byte
	_ = json.Encode(&config.AssignRoleRequest{UserID: "u-123", RoleCode: "client-role"}, &body2)
	ctxAssign.InBody = body2
	hAssign(ctxAssign)
	if ctxAssign.Status != 200 {
		t.Fatalf("AssignRole status %d, want 200", ctxAssign.Status)
	}

	entry2 := lastAudit(t, backend.DB)
	if entry2.Action != config.AuditRoleGrantedByClient {
		t.Errorf("action: got %q, want %q", entry2.Action, config.AuditRoleGrantedByClient)
	}
	if entry2.Target != "u-123:client-role" {
		t.Errorf("target: got %q, want %q", entry2.Target, "u-123:client-role")
	}
}

// TestAudit_TruncatesHostileDetail: un Origin de 10 000 bytes se guarda
// truncado a auditDetailMax.
func TestAudit_TruncatesHostileDetail(t *testing.T) {
	b, r := setupPanel(t, "admin@example.com")

	ctx := adminCtx(t, b, "admin@example.com")
	ctx.InMethod = "POST"
	ctx.InPath = config.PathAdminProjects
	ctx.SetHeader("Sec-Fetch-Site", "cross-site")
	ctx.SetHeader("Origin", strings.Repeat("x", 10000))
	ctx.InBody = encodeBody(t, &config.AdminCreateProjectRequest{Name: "Huge Origin"})
	r.Invoke("POST", config.PathAdminProjects, ctx)
	if ctx.Status != 403 {
		t.Fatalf("status %d, want 403", ctx.Status)
	}

	entry := lastAudit(t, b.DB)
	if entry.Action != config.AuditOriginDenied {
		t.Fatalf("action: got %q, want %q", entry.Action, config.AuditOriginDenied)
	}
	if len(entry.Detail) > 200 {
		t.Errorf("detail length %d, want at most 200", len(entry.Detail))
	}
}
