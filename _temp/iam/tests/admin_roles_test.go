//go:build !wasm

package tests

import (
	"testing"

	"webtyp.com/json"
	"webtyp.com/model"
	"webtyp.com/router/mock"

	"github.com/veltylabs/iam/config"
)

func encodeBody(t *testing.T, v model.Encodable) []byte {
	t.Helper()
	var out []byte
	if err := json.Encode(v, &out); err != nil {
		t.Fatalf("encodeBody: %v", err)
	}
	return out
}

func TestCreateRole_ThenListed(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	u, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")
	_ = config.CreateProject(b.DB, "proj1", "App 1", "")

	hCreate := config.CreateRoleHandler(b.DB, b.RBAC, b.IDs, b.Auth)
	ctx := &mock.Context{InMethod: "POST", InPath: config.PathAdminRoles}
	ctx.SetUserID(u.Id)
	ctx.InBody = encodeBody(t, &config.AdminCreateRoleRequest{ProjectId: "proj1", Code: "admin", Name: "Administrator", Description: "Desc"})

	hCreate(ctx)
	if ctx.Status != 201 {
		t.Fatalf("CreateRole status %d, want 201", ctx.Status)
	}

	hList := config.ListRolesHandler(b.DB, b.RBAC)
	ctxList := &mock.Context{InMethod: "GET", InPath: config.PathAdminRoles + "?project_id=proj1"}
	ctxList.SetUserID(u.Id)
	hList(ctxList)

	if ctxList.Status != 200 {
		t.Fatalf("ListRoles status %d, want 200", ctxList.Status)
	}

	var rolesResp config.AdminRolesResponse
	_ = json.Decode(ctxList.ResponseBody(), &rolesResp)
	if len(rolesResp.Roles) != 1 || rolesResp.Roles[0].Code != "admin" {
		t.Fatalf("unexpected roles response: %+v", rolesResp.Roles)
	}
}

func TestSetRoleTTL(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	u, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")
	_ = config.CreateProject(b.DB, "proj1", "App 1", "")
	_ = b.RBAC.CreateRole("proj1", "role1", model.RoleCode("editor"), "Editor", "")

	hTTL := config.SetRoleTTLHandler(b.DB, b.RBAC, b.IDs, b.Auth)
	ctx := &mock.Context{InMethod: "POST", InPath: config.PathAdminRoleTTL}
	ctx.SetUserID(u.Id)
	ctx.InBody = encodeBody(t, &config.AdminRoleTTLRequest{ProjectId: "proj1", Code: "editor", SessionTtl: 600})

	hTTL(ctx)
	if ctx.Status != 200 {
		t.Fatalf("SetRoleTTL status %d, want 200", ctx.Status)
	}

	role, err := b.RBAC.GetRoleByCode("proj1", model.RoleCode("editor"))
	if err != nil || role.SessionTtl != 600 {
		t.Fatalf("SessionTTL not updated: got %d, want 600 (err=%v)", role.SessionTtl, err)
	}
}

func TestDeleteRole(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	u, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")
	_ = config.CreateProject(b.DB, "proj1", "App 1", "")
	_ = b.RBAC.CreateRole("proj1", "role1", model.RoleCode("editor"), "Editor", "")

	hDelete := config.DeleteRoleHandler(b.DB, b.RBAC, b.IDs, b.Auth)
	ctx := &mock.Context{InMethod: "POST", InPath: config.PathAdminRoleDelete}
	ctx.SetUserID(u.Id)
	ctx.InBody = encodeBody(t, &config.AdminRoleRefRequest{ProjectId: "proj1", Code: "editor"})

	hDelete(ctx)
	if ctx.Status != 200 {
		t.Fatalf("DeleteRole status %d, want 200", ctx.Status)
	}

	_, err := b.RBAC.GetRoleByCode("proj1", model.RoleCode("editor"))
	if err == nil {
		t.Fatalf("role editor still exists after deletion")
	}

	ctx404 := &mock.Context{InMethod: "POST", InPath: config.PathAdminRoleDelete}
	ctx404.SetUserID(u.Id)
	ctx404.InBody = encodeBody(t, &config.AdminRoleRefRequest{ProjectId: "proj1", Code: "editor"})
	hDelete(ctx404)
	if ctx404.Status != 404 {
		t.Fatalf("DeleteRole non-existent role status %d, want 404", ctx404.Status)
	}
}

func TestRolesIsolatedByProject(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	u, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")

	_ = config.CreateProject(b.DB, "proj1", "App 1", "")
	_ = config.CreateProject(b.DB, "proj2", "App 2", "")

	_ = b.RBAC.CreateRole("proj1", "r1", model.RoleCode("admin"), "Admin 1", "")
	_ = b.RBAC.CreateRole("proj2", "r2", model.RoleCode("admin"), "Admin 2", "")

	hList := config.ListRolesHandler(b.DB, b.RBAC)
	ctxList := &mock.Context{InMethod: "GET", InPath: config.PathAdminRoles + "?project_id=proj1"}
	ctxList.SetUserID(u.Id)
	hList(ctxList)

	var resp config.AdminRolesResponse
	_ = json.Decode(ctxList.ResponseBody(), &resp)
	if len(resp.Roles) != 1 || resp.Roles[0].Name != "Admin 1" {
		t.Fatalf("roles not isolated by project: %+v", resp.Roles)
	}
}

func TestCreateRoleDuplicateCode409(t *testing.T) {
	b, _ := setupPanel(t, "admin@example.com")
	u, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")
	_ = config.CreateProject(b.DB, "proj1", "App 1", "")

	hCreate := config.CreateRoleHandler(b.DB, b.RBAC, b.IDs, b.Auth)
	call := func() int {
		ctx := &mock.Context{InMethod: "POST", InPath: config.PathAdminRoles}
		ctx.SetUserID(u.Id)
		ctx.InBody = encodeBody(t, &config.AdminCreateRoleRequest{ProjectId: "proj1", Code: "editor", Name: "Editor", Description: ""})
		hCreate(ctx)
		return ctx.Status
	}
	if status := call(); status != 201 {
		t.Fatalf("first CreateRole status %d, want 201", status)
	}
	if status := call(); status != 409 {
		t.Fatalf("second CreateRole with duplicate code status %d, want 409", status)
	}
}
