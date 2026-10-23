package config

import (
	"webtyp.com/auth/authority"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/rbac"
)

const (
	// IAMProjectID is iam's own project in rbac: the roles below live here.
	IAMProjectID = "iam"

	RoleIAMAdmin  model.RoleCode = "iam_admin"  // humans who run the panel
	RoleIAMClient model.RoleCode = "iam_client" // consumer apps (machine principals)

	ResourceProject   model.Resource = "iam_project"
	ResourceRole      model.Resource = "iam_role"
	ResourceUserRole  model.Resource = "iam_user_role"
	ResourceAudit     model.Resource = "iam_audit"
	ResourceRoutes    model.Resource = "iam_routes"
	ResourceHealth    model.Resource = "iam_health"
	ResourceToken     model.Resource = "iam_token"
	ResourceIdentity  model.Resource = "iam_identity"
	ResourceRoleGrant model.Resource = "iam_role_grant"
)

const (
	RoleIAMAdminID  = "role_iam_admin"
	RoleIAMClientID = "role_iam_client"
)

// Authorizer answers router's Authorize for iam's own routes.
func Authorizer(rbacSvc *rbac.Service) model.Authorizer {
	return func(userID string, r model.Resource, a model.Action) bool {
		return rbacSvc.Can(IAMProjectID, userID, r, a)
	}
}

type permDef struct {
	id       string
	name     string
	resource model.Resource
	actions  model.Action
}

var adminPermDefs = [...]permDef{
	{id: "perm_iam_admin_project", name: "Manage projects", resource: ResourceProject, actions: model.Create | model.Read | model.Update},
	{id: "perm_iam_admin_role", name: "Manage roles", resource: ResourceRole, actions: model.Create | model.Read | model.Update | model.Delete},
	{id: "perm_iam_admin_user_role", name: "Manage user roles", resource: ResourceUserRole, actions: model.Create | model.Read | model.Delete},
	{id: "perm_iam_admin_audit", name: "Read audit", resource: ResourceAudit, actions: model.Read},
	{id: "perm_iam_admin_routes", name: "Read routes", resource: ResourceRoutes, actions: model.Read},
	{id: "perm_iam_admin_health", name: "Read health", resource: ResourceHealth, actions: model.Read},
}

var clientPermDefs = [...]permDef{
	{id: "perm_iam_client_token", name: "Issue auth token", resource: ResourceToken, actions: model.Create},
	{id: "perm_iam_client_identity", name: "Resolve identity", resource: ResourceIdentity, actions: model.Create},
	{id: "perm_iam_client_role_grant", name: "Assign role", resource: ResourceRoleGrant, actions: model.Create},
}

// EnsurePolicy creates both roles and their permissions idempotently, and
// grants RoleIAMAdmin to every email in adminEmails (creating the user if
// needed, via the existing EnsureRole). IAM_ADMIN_EMAILS is a SEED: removing
// an email does not revoke. Revocation is done in the panel and audited.
//
// Where it runs: in NewProductionBackend and the local backend, right after
// auth/rbac are built. Why not in cmd/migrate: goflare.NewD1Migrator
// returns a ddl.Execer that runs DDL only and cannot run the orm queries
// seeding needs. The cost is ~2 + 2N D1 queries per isolate cold start
// (N = admin emails). They are idempotent and data, not schema.
func EnsurePolicy(rbacSvc *rbac.Service, authMod *authority.Module, adminEmails []string) error {
	if rbacSvc == nil {
		return fmt.Err("bootstrap: rbac service is nil")
	}
	if authMod == nil {
		return fmt.Err("bootstrap: authority module is nil")
	}

	// 1. Ensure RoleIAMAdmin and assign adminEmails
	if err := EnsureRole(rbacSvc, authMod, IAMProjectID, RoleIAMAdminID, RoleIAMAdmin, "IAM Admin", "Administrator of IAM", adminEmails); err != nil {
		return err
	}

	// 2. Ensure RoleIAMAdmin permissions
	for i := 0; i < len(adminPermDefs); i++ {
		p := adminPermDefs[i]
		if err := rbacSvc.CreatePermission(IAMProjectID, p.id, p.name, p.resource, p.actions); err != nil && !isAlreadyExistsErr(err) {
			return err
		}
		if err := rbacSvc.AssignPermission(IAMProjectID, RoleIAMAdminID, p.id); err != nil && !isAlreadyExistsErr(err) {
			return err
		}
	}

	// 3. Ensure RoleIAMClient
	if err := rbacSvc.CreateRole(IAMProjectID, RoleIAMClientID, RoleIAMClient, "IAM Client", "Consumer apps machine principals"); err != nil && !isAlreadyExistsErr(err) {
		return err
	}

	// 4. Ensure RoleIAMClient permissions
	for i := 0; i < len(clientPermDefs); i++ {
		p := clientPermDefs[i]
		if err := rbacSvc.CreatePermission(IAMProjectID, p.id, p.name, p.resource, p.actions); err != nil && !isAlreadyExistsErr(err) {
			return err
		}
		if err := rbacSvc.AssignPermission(IAMProjectID, RoleIAMClientID, p.id); err != nil && !isAlreadyExistsErr(err) {
			return err
		}
	}

	return nil
}
