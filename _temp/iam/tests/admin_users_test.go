//go:build !wasm

package tests

import (
	"testing"

	"webtyp.com/json"
	"webtyp.com/model"
	"webtyp.com/router/mock"

	"github.com/veltylabs/iam/config"
)

func TestAssignRole_CreatesUserByEmail(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	adminU, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")
	_ = config.CreateProject(b.DB, "proj1", "App 1", "")
	_ = b.RBAC.CreateRole("proj1", "r1", model.RoleCode("editor"), "Editor", "")

	hAssign := config.AssignUserHandler(b.DB, b.Auth, b.RBAC, b.IDs)
	ctx := &mock.Context{InMethod: "POST", InPath: config.PathAdminUserAssign}
	ctx.SetUserID(adminU.Id)
	ctx.InBody = encodeBody(t, &config.AdminUserRoleRequest{ProjectId: "proj1", Code: "editor", Email: "newuser@example.com"})

	hAssign(ctx)
	if ctx.Status != 200 {
		t.Fatalf("AssignUser status %d, want 200", ctx.Status)
	}

	var assignResp config.AdminAssignResponse
	_ = json.Decode(ctx.ResponseBody(), &assignResp)
	if assignResp.Sub == "" {
		t.Fatalf("empty sub in assign response")
	}

	u, err := b.Auth.UserByEmail("newuser@example.com")
	if err != nil || u.Id != assignResp.Sub {
		t.Fatalf("user not created properly: %+v, err=%v", u, err)
	}

	ctx2 := &mock.Context{InMethod: "POST", InPath: config.PathAdminUserAssign}
	ctx2.SetUserID(adminU.Id)
	ctx2.InBody = encodeBody(t, &config.AdminUserRoleRequest{ProjectId: "proj1", Code: "editor", Email: "newuser@example.com"})
	hAssign(ctx2)
	if ctx2.Status != 200 {
		t.Fatalf("AssignUser second call status %d, want 200", ctx2.Status)
	}

	var assignResp2 config.AdminAssignResponse
	_ = json.Decode(ctx2.ResponseBody(), &assignResp2)
	if assignResp2.Sub != assignResp.Sub {
		t.Fatalf("idempotent assign created duplicate user sub: %s vs %s", assignResp2.Sub, assignResp.Sub)
	}
}

func TestAssignRole_UnknownRole404(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	adminU, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")
	_ = config.CreateProject(b.DB, "proj1", "App 1", "")

	hAssign := config.AssignUserHandler(b.DB, b.Auth, b.RBAC, b.IDs)
	ctx := &mock.Context{InMethod: "POST", InPath: config.PathAdminUserAssign}
	ctx.SetUserID(adminU.Id)
	ctx.InBody = encodeBody(t, &config.AdminUserRoleRequest{ProjectId: "proj1", Code: "unknown", Email: "user@example.com"})

	hAssign(ctx)
	if ctx.Status != 404 {
		t.Fatalf("AssignUser unknown role status %d, want 404", ctx.Status)
	}
}

func TestRevokeRole(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	adminU, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")
	_ = config.CreateProject(b.DB, "proj1", "App 1", "")
	_ = b.RBAC.CreateRole("proj1", "r1", model.RoleCode("editor"), "Editor", "")

	targetU, _ := b.Auth.CreateUser("target@example.com", "Target", "")
	role, _ := b.RBAC.GetRoleByCode("proj1", model.RoleCode("editor"))
	_ = b.RBAC.AssignRole("proj1", targetU.Id, role.Id)

	hRevoke := config.RevokeUserHandler(b.DB, b.Auth, b.RBAC, b.IDs)
	ctx := &mock.Context{InMethod: "POST", InPath: config.PathAdminUserRevoke}
	ctx.SetUserID(adminU.Id)
	ctx.InBody = encodeBody(t, &config.AdminUserRoleRequest{ProjectId: "proj1", Code: "editor", Email: "target@example.com"})

	hRevoke(ctx)
	if ctx.Status != 200 {
		t.Fatalf("RevokeUser status %d, want 200", ctx.Status)
	}

	hUsers := config.ListRoleUsersHandler(b.DB, b.Auth, b.RBAC)
	ctxList := &mock.Context{InMethod: "GET", InPath: config.PathAdminRoleUsers + "?project_id=proj1&code=editor"}
	ctxList.SetUserID(adminU.Id)
	hUsers(ctxList)

	var usersResp config.AdminRoleUsersResponse
	_ = json.Decode(ctxList.ResponseBody(), &usersResp)
	if len(usersResp.Users) != 0 {
		t.Fatalf("user still assigned after revocation: %+v", usersResp.Users)
	}
}

func TestAssignRole_SubMatchesLoginSub(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	adminU, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")
	_ = config.CreateProject(b.DB, "proj1", "App 1", "")
	_ = b.RBAC.CreateRole("proj1", "r1", model.RoleCode("editor"), "Editor", "")

	hAssign := config.AssignUserHandler(b.DB, b.Auth, b.RBAC, b.IDs)
	ctx := &mock.Context{InMethod: "POST", InPath: config.PathAdminUserAssign}
	ctx.SetUserID(adminU.Id)
	ctx.InBody = encodeBody(t, &config.AdminUserRoleRequest{ProjectId: "proj1", Code: "editor", Email: "someone@example.com"})

	hAssign(ctx)
	var assignResp config.AdminAssignResponse
	_ = json.Decode(ctx.ResponseBody(), &assignResp)

	u, _ := b.Auth.UserByEmail("someone@example.com")
	if u.Id != assignResp.Sub {
		t.Fatalf("assigned sub %s != UserByEmail sub %s", assignResp.Sub, u.Id)
	}
}

func TestRevokeInvalidatesRBACCache(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	adminU, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")
	_ = config.CreateProject(b.DB, "proj-cache", "Cache App", "")
	_ = b.RBAC.CreateRole("proj-cache", "r1", model.RoleCode("editor"), "Editor", "")
	_ = b.RBAC.CreatePermission("proj-cache", "p1", "Read docs", "docs", model.Read)
	_ = b.RBAC.AssignPermission("proj-cache", "r1", "p1")

	hAssign := config.AssignUserHandler(b.DB, b.Auth, b.RBAC, b.IDs)
	ctxAssign := &mock.Context{InMethod: "POST", InPath: config.PathAdminUserAssign}
	ctxAssign.SetUserID(adminU.Id)
	ctxAssign.InBody = encodeBody(t, &config.AdminUserRoleRequest{ProjectId: "proj-cache", Code: "editor", Email: "cached@example.com"})
	hAssign(ctxAssign)
	if ctxAssign.Status != 200 {
		t.Fatalf("AssignUser status %d, want 200", ctxAssign.Status)
	}

	target, err := b.Auth.UserByEmail("cached@example.com")
	if err != nil {
		t.Fatalf("UserByEmail: %v", err)
	}
	if !b.RBAC.Can("proj-cache", target.Id, "docs", model.Read) {
		t.Fatal("expected permission granted after assign")
	}

	hRevoke := config.RevokeUserHandler(b.DB, b.Auth, b.RBAC, b.IDs)
	ctxRevoke := &mock.Context{InMethod: "POST", InPath: config.PathAdminUserRevoke}
	ctxRevoke.SetUserID(adminU.Id)
	ctxRevoke.InBody = encodeBody(t, &config.AdminUserRoleRequest{ProjectId: "proj-cache", Code: "editor", Email: "cached@example.com"})
	hRevoke(ctxRevoke)
	if ctxRevoke.Status != 200 {
		t.Fatalf("RevokeUser status %d, want 200", ctxRevoke.Status)
	}
	if b.RBAC.Can("proj-cache", target.Id, "docs", model.Read) {
		t.Fatal("revoked permission still granted: RevokeUserHandler did not invalidate the RBAC cache")
	}
}
