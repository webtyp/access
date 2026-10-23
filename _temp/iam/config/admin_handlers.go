package config

import (
	"webtyp.com/auth/authority"
	"webtyp.com/auth/passkey"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/rbac"
	"webtyp.com/router"
)

func adminEmailOf(authMod *authority.Module, ctx router.Context) string {
	if uid := ctx.UserID(); uid != "" {
		if u, err := authMod.GetUser(uid); err == nil {
			return u.Email
		}
	}
	return ""
}

// Me responde con datos del administrador actual.
func Me(db *orm.DB, authMod *authority.Module, rbacSvc *rbac.Service) router.HandlerFunc {
	return func(ctx router.Context) {
		uid := ctx.UserID()
		if uid == "" {
			ctx.WriteStatus(403)
			return
		}
		u, err := authMod.GetUser(uid)
		if err != nil {
			ctx.WriteStatus(500)
			return
		}

		isAdmin := rbacSvc.Can(IAMProjectID, uid, ResourceProject, model.Read)

		qb := db.Query(&passkey.PasskeyCredential{}).Where(passkey.PasskeyCredential_.UserId).Eq(uid)
		creds, err := passkey.ReadAllPasskeyCredential(qb)
		hasPasskey := err == nil && len(creds) > 0

		multiFactor := ctx.Assurance() == model.AssuranceMultiFactor

		ctx.WriteStatus(200)
		_ = ctx.Encode(&AdminMeResponse{
			Email:       u.Email,
			Name:        u.Name,
			IsAdmin:     isAdmin,
			HasPasskey:  hasPasskey,
			MultiFactor: multiFactor,
		})
	}
}

// ListProjectsHandler lista todos los proyectos.
func ListProjectsHandler(db *orm.DB) router.HandlerFunc {
	return func(ctx router.Context) {
		projects, err := ListProjects(db)
		if err != nil {
			ctx.WriteStatus(500)
			return
		}
		views := make([]AdminProjectView, 0, len(projects))
		for _, p := range projects {
			views = append(views, AdminProjectView{
				Id:        p.Id,
				Name:      p.Name,
				Active:    p.Active != 0,
				CreatedAt: int64(p.CreatedAt),
			})
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&AdminProjectsResponse{Projects: views})
	}
}

// CreateProjectHandler crea un nuevo proyecto.
func CreateProjectHandler(db *orm.DB, ids model.IDGenerator, authMod *authority.Module, rbacSvc *rbac.Service) router.HandlerFunc {
	return func(ctx router.Context) {
		var req AdminCreateProjectRequest
		if err := ctx.Decode(&req); err != nil || fmt.TrimSpace(req.Name) == "" {
			ctx.WriteStatus(400)
			return
		}
		id := ids.NewID()
		if err := CreateProject(db, id, req.Name, req.PublicKey); err != nil {
			ctx.WriteStatus(400)
			return
		}

		_ = rbacSvc.AssignRoleByCode(IAMProjectID, id, RoleIAMClient)

		adminEmail := adminEmailOf(authMod, ctx)
		if err := RecordAudit(db, ids, adminEmail, AuditProjectCreate, id, req.Name); err != nil {
			fmt.Println("audit:", err)
		}
		ctx.WriteStatus(201)
		_ = ctx.Encode(&AdminProjectIDRequest{
			ProjectId: id,
		})
	}
}

// SetProjectKeyHandler replaces a project's public key.
func SetProjectKeyHandler(db *orm.DB, ids model.IDGenerator, authMod *authority.Module) router.HandlerFunc {
	return func(ctx router.Context) {
		var req AdminProjectKeyRequest
		if err := ctx.Decode(&req); err != nil || fmt.TrimSpace(req.ProjectId) == "" {
			ctx.WriteStatus(400)
			return
		}
		if err := SetProjectKey(db, req.ProjectId, req.PublicKey); err != nil {
			if IsProjectNotFound(err) {
				ctx.WriteStatus(404)
				return
			}
			ctx.WriteStatus(400)
			return
		}
		adminEmail := adminEmailOf(authMod, ctx)
		if err := RecordAudit(db, ids, adminEmail, AuditProjectRotate, req.ProjectId, ""); err != nil {
			fmt.Println("audit:", err)
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&AdminProjectIDRequest{
			ProjectId: req.ProjectId,
		})
	}
}

// SetActiveHandler activa o desactiva un proyecto.
func SetActiveHandler(db *orm.DB, ids model.IDGenerator, authMod *authority.Module) router.HandlerFunc {
	return func(ctx router.Context) {
		var req AdminSetActiveRequest
		if err := ctx.Decode(&req); err != nil || fmt.TrimSpace(req.ProjectId) == "" {
			ctx.WriteStatus(400)
			return
		}
		if err := SetProjectActive(db, req.ProjectId, req.Active); err != nil {
			if IsProjectNotFound(err) {
				ctx.WriteStatus(404)
				return
			}
			ctx.WriteStatus(500)
			return
		}
		action := AuditProjectActivate
		if !req.Active {
			action = AuditProjectDeactivate
		}
		adminEmail := adminEmailOf(authMod, ctx)
		if err := RecordAudit(db, ids, adminEmail, action, req.ProjectId, ""); err != nil {
			fmt.Println("audit:", err)
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&AdminProjectIDRequest{ProjectId: req.ProjectId})
	}
}

// ListRolesHandler lista los roles de un proyecto dado por query param ?project_id=.
func ListRolesHandler(db *orm.DB, rbacSvc *rbac.Service) router.HandlerFunc {
	return func(ctx router.Context) {
		projectID, ok := router.QueryParam(ctx.Path(), "project_id")
		if !ok || fmt.TrimSpace(projectID) == "" {
			ctx.WriteStatus(400)
			return
		}
		roles, err := ListRoles(db, rbacSvc, projectID)
		if err != nil {
			ctx.WriteStatus(500)
			return
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&AdminRolesResponse{Roles: roles})
	}
}

// CreateRoleHandler crea un rol dentro de un proyecto.
func CreateRoleHandler(db *orm.DB, rbacSvc *rbac.Service, ids model.IDGenerator, authMod *authority.Module) router.HandlerFunc {
	return func(ctx router.Context) {
		var req AdminCreateRoleRequest
		if err := ctx.Decode(&req); err != nil || fmt.TrimSpace(req.ProjectId) == "" || fmt.TrimSpace(req.Code) == "" {
			ctx.WriteStatus(400)
			return
		}
		roleID := ids.NewID()
		if err := rbacSvc.CreateRole(req.ProjectId, roleID, model.RoleCode(req.Code), req.Name, req.Description); err != nil {
			if rbac.IsDuplicateRoleCode(err) {
				ctx.WriteStatus(409)
				return
			}
			ctx.WriteStatus(500)
			return
		}
		target := req.ProjectId + "/" + req.Code
		adminEmail := adminEmailOf(authMod, ctx)
		if err := RecordAudit(db, ids, adminEmail, AuditRoleCreate, target, req.Name); err != nil {
			fmt.Println("audit:", err)
		}
		ctx.WriteStatus(201)
		_ = ctx.Encode(&AdminRoleRefRequest{ProjectId: req.ProjectId, Code: req.Code})
	}
}

// SetRoleTTLHandler actualiza el SessionTTL de un rol.
func SetRoleTTLHandler(db *orm.DB, rbacSvc *rbac.Service, ids model.IDGenerator, authMod *authority.Module) router.HandlerFunc {
	return func(ctx router.Context) {
		var req AdminRoleTTLRequest
		if err := ctx.Decode(&req); err != nil || fmt.TrimSpace(req.ProjectId) == "" || fmt.TrimSpace(req.Code) == "" {
			ctx.WriteStatus(400)
			return
		}
		role, err := rbacSvc.GetRoleByCode(req.ProjectId, model.RoleCode(req.Code))
		if err != nil {
			if rbac.IsRoleNotFound(err) {
				ctx.WriteStatus(404)
				return
			}
			ctx.WriteStatus(500)
			return
		}
		if err := rbacSvc.SetRoleSessionTTL(req.ProjectId, role.Id, req.SessionTtl); err != nil {
			ctx.WriteStatus(500)
			return
		}
		target := req.ProjectId + "/" + req.Code
		detail := fmt.Sprintf("%d", req.SessionTtl)
		adminEmail := adminEmailOf(authMod, ctx)
		if err := RecordAudit(db, ids, adminEmail, AuditRoleSetTTL, target, detail); err != nil {
			fmt.Println("audit:", err)
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&AdminRoleRefRequest{ProjectId: req.ProjectId, Code: req.Code})
	}
}

// DeleteRoleHandler elimina un rol de un proyecto.
func DeleteRoleHandler(db *orm.DB, rbacSvc *rbac.Service, ids model.IDGenerator, authMod *authority.Module) router.HandlerFunc {
	return func(ctx router.Context) {
		var req AdminRoleRefRequest
		if err := ctx.Decode(&req); err != nil || fmt.TrimSpace(req.ProjectId) == "" || fmt.TrimSpace(req.Code) == "" {
			ctx.WriteStatus(400)
			return
		}
		if err := rbacSvc.DeleteRoleByCode(req.ProjectId, model.RoleCode(req.Code)); err != nil {
			if rbac.IsRoleNotFound(err) {
				ctx.WriteStatus(404)
				return
			}
			ctx.WriteStatus(500)
			return
		}
		target := req.ProjectId + "/" + req.Code
		adminEmail := adminEmailOf(authMod, ctx)
		if err := RecordAudit(db, ids, adminEmail, AuditRoleDelete, target, ""); err != nil {
			fmt.Println("audit:", err)
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&AdminRoleRefRequest{ProjectId: req.ProjectId, Code: req.Code})
	}
}

// ListRoleUsersHandler lista usuarios asignados a un rol (?project_id=&code=).
func ListRoleUsersHandler(db *orm.DB, authMod *authority.Module, rbacSvc *rbac.Service) router.HandlerFunc {
	return func(ctx router.Context) {
		projectID, ok := router.QueryParam(ctx.Path(), "project_id")
		if !ok || fmt.TrimSpace(projectID) == "" {
			ctx.WriteStatus(400)
			return
		}
		code, ok := router.QueryParam(ctx.Path(), "code")
		if !ok || fmt.TrimSpace(code) == "" {
			ctx.WriteStatus(400)
			return
		}
		users, err := ListRoleUsers(db, authMod, rbacSvc, projectID, code)
		if err != nil {
			if rbac.IsRoleNotFound(err) {
				ctx.WriteStatus(404)
				return
			}
			ctx.WriteStatus(500)
			return
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&AdminRoleUsersResponse{Users: users})
	}
}

// AssignUserHandler asigna un rol a un usuario por email (lo crea si no existe).
func AssignUserHandler(db *orm.DB, authMod *authority.Module, rbacSvc *rbac.Service, ids model.IDGenerator) router.HandlerFunc {
	return func(ctx router.Context) {
		var req AdminUserRoleRequest
		if err := ctx.Decode(&req); err != nil || fmt.TrimSpace(req.ProjectId) == "" || fmt.TrimSpace(req.Code) == "" || fmt.TrimSpace(req.Email) == "" {
			ctx.WriteStatus(400)
			return
		}
		u, err := authMod.UserByEmail(req.Email)
		if err != nil {
			u, err = authMod.CreateUser(req.Email, req.Email, "")
			if err != nil {
				ctx.WriteStatus(500)
				return
			}
		}
		if err := rbacSvc.AssignRoleByCode(req.ProjectId, u.Id, model.RoleCode(req.Code)); err != nil {
			if rbac.IsRoleNotFound(err) {
				ctx.WriteStatus(404)
				return
			}
			ctx.WriteStatus(500)
			return
		}
		target := req.ProjectId + "/" + req.Code
		adminEmail := adminEmailOf(authMod, ctx)
		if err := RecordAudit(db, ids, adminEmail, AuditUserAssign, target, req.Email); err != nil {
			fmt.Println("audit:", err)
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&AdminAssignResponse{Sub: u.Id})
	}
}

// RevokeUserHandler revoca un rol a un usuario.
func RevokeUserHandler(db *orm.DB, authMod *authority.Module, rbacSvc *rbac.Service, ids model.IDGenerator) router.HandlerFunc {
	return func(ctx router.Context) {
		var req AdminUserRoleRequest
		if err := ctx.Decode(&req); err != nil || fmt.TrimSpace(req.ProjectId) == "" || fmt.TrimSpace(req.Code) == "" || fmt.TrimSpace(req.Email) == "" {
			ctx.WriteStatus(400)
			return
		}
		u, err := authMod.UserByEmail(req.Email)
		if err != nil {
			ctx.WriteStatus(404)
			return
		}
		if err := rbacSvc.RevokeRoleByCode(req.ProjectId, u.Id, model.RoleCode(req.Code)); err != nil {
			if rbac.IsRoleNotFound(err) {
				ctx.WriteStatus(404)
				return
			}
			ctx.WriteStatus(500)
			return
		}
		target := req.ProjectId + "/" + req.Code
		adminEmail := adminEmailOf(authMod, ctx)
		if err := RecordAudit(db, ids, adminEmail, AuditUserRevoke, target, req.Email); err != nil {
			fmt.Println("audit:", err)
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&AdminAssignResponse{Sub: u.Id})
	}
}

// ListAuditHandler lista las entradas del registro de auditoría.
func ListAuditHandler(db *orm.DB) router.HandlerFunc {
	return func(ctx router.Context) {
		entries, err := ListAudit(db, 200)
		if err != nil {
			ctx.WriteStatus(500)
			return
		}
		views := make([]AdminAuditEntry, 0, len(entries))
		for _, e := range entries {
			views = append(views, AdminAuditEntry{
				ActorEmail: e.ActorEmail,
				Action:     e.Action,
				Target:     e.Target,
				Detail:     e.Detail,
				CreatedAt:  int64(e.CreatedAt),
			})
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&AdminAuditResponse{Entries: views})
	}
}
