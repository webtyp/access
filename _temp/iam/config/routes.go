package config

import (
	"webtyp.com/auth/authority"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/rbac"
	"webtyp.com/router"
	"webtyp.com/router/security"
)

const (
	PathHealth       = "/api/health"
	PathHealthDB     = "/api/health/db"
	PathToken        = "/api/token"
	PathUsersResolve = "/api/users/resolve"
	PathRolesAssign  = "/api/roles/assign"

	PathAdminMe            = "/api/admin/me"
	PathAdminProjects      = "/api/admin/projects"     // GET lista · POST crea
	PathAdminProjectKey    = "/api/admin/projects/key" // POST
	PathAdminProjectActive = "/api/admin/projects/active" // POST
	PathAdminRoles         = "/api/admin/roles"        // GET ?project_id= · POST crea
	PathAdminRoleTTL       = "/api/admin/roles/ttl"    // POST
	PathAdminRoleDelete    = "/api/admin/roles/delete" // POST
	PathAdminRoleUsers     = "/api/admin/roles/users"  // GET ?project_id=&code=
	PathAdminUserAssign    = "/api/admin/users/assign" // POST
	PathAdminUserRevoke    = "/api/admin/users/revoke" // POST
	PathAdminAudit         = "/api/admin/audit"        // GET

	bodyHealthOK   = `{"ok":true}`
	bodyHealthFail = `{"ok":false}`
)

// RegisterRoutes monta todas las rutas del servicio iam con sus dependencias explícitas.
func RegisterRoutes(r router.Router, b *Backend, adminEmails []string) {
	policy := security.Policy{}.AllowImages("https://lh3.googleusercontent.com")
	r.Use(policy.Middleware())

	b.Auth.MountAPI(r)

	r.Get(PathHealth, Health()).Public()
	r.Get(PathHealthDB, HealthDB(b.DB)).Requires(ResourceHealth, model.Read).RequiresMFA()
	r.Post(PathToken, Token(b.DB, b.Auth, b.RBAC, b.JWTSecret, b.IDs)).Requires(ResourceToken, model.Create)
	r.Post(PathUsersResolve, ResolveUser(b.DB, b.Auth, b.IDs)).Requires(ResourceIdentity, model.Create)
	r.Post(PathRolesAssign, AssignRole(b.DB, b.RBAC, b.IDs)).Requires(ResourceRoleGrant, model.Create)

	r.Get(PathAdminMe, Me(b.DB, b.Auth, b.RBAC)).Authenticated()
	r.Get(PathAdminProjects, ListProjectsHandler(b.DB)).Requires(ResourceProject, model.Read).RequiresMFA()
	r.Post(PathAdminProjects, RequireSameOrigin(b.DB, b.IDs, b.Auth, b.Origin, CreateProjectHandler(b.DB, b.IDs, b.Auth, b.RBAC))).Requires(ResourceProject, model.Create).RequiresMFA()
	r.Post(PathAdminProjectKey, RequireSameOrigin(b.DB, b.IDs, b.Auth, b.Origin, SetProjectKeyHandler(b.DB, b.IDs, b.Auth))).Requires(ResourceProject, model.Update).RequiresMFA()
	r.Post(PathAdminProjectActive, RequireSameOrigin(b.DB, b.IDs, b.Auth, b.Origin, SetActiveHandler(b.DB, b.IDs, b.Auth))).Requires(ResourceProject, model.Update).RequiresMFA()
	r.Get(PathAdminRoles, ListRolesHandler(b.DB, b.RBAC)).Requires(ResourceRole, model.Read).RequiresMFA()
	r.Post(PathAdminRoles, RequireSameOrigin(b.DB, b.IDs, b.Auth, b.Origin, CreateRoleHandler(b.DB, b.RBAC, b.IDs, b.Auth))).Requires(ResourceRole, model.Create).RequiresMFA()
	r.Post(PathAdminRoleTTL, RequireSameOrigin(b.DB, b.IDs, b.Auth, b.Origin, SetRoleTTLHandler(b.DB, b.RBAC, b.IDs, b.Auth))).Requires(ResourceRole, model.Update).RequiresMFA()
	r.Post(PathAdminRoleDelete, RequireSameOrigin(b.DB, b.IDs, b.Auth, b.Origin, DeleteRoleHandler(b.DB, b.RBAC, b.IDs, b.Auth))).Requires(ResourceRole, model.Delete).RequiresMFA()
	r.Get(PathAdminRoleUsers, ListRoleUsersHandler(b.DB, b.Auth, b.RBAC)).Requires(ResourceUserRole, model.Read).RequiresMFA()
	r.Post(PathAdminUserAssign, RequireSameOrigin(b.DB, b.IDs, b.Auth, b.Origin, AssignUserHandler(b.DB, b.Auth, b.RBAC, b.IDs))).Requires(ResourceUserRole, model.Create).RequiresMFA()
	r.Post(PathAdminUserRevoke, RequireSameOrigin(b.DB, b.IDs, b.Auth, b.Origin, RevokeUserHandler(b.DB, b.Auth, b.RBAC, b.IDs))).Requires(ResourceUserRole, model.Delete).RequiresMFA()
	r.Get(PathAdminAudit, ListAuditHandler(b.DB)).Requires(ResourceAudit, model.Read).RequiresMFA()

	router.MountIntrospection(r, router.IntrospectionPath, nil).Requires(ResourceRoutes, model.Read).RequiresMFA()
}

// Health responde si el servidor está vivo.
func Health() router.HandlerFunc {
	return func(ctx router.Context) {
		ctx.SetHeader("Content-Type", "application/json")
		ctx.WriteStatus(200)
		_, _ = ctx.Write([]byte(bodyHealthOK))
	}
}

// HealthDB comprueba que la base responde.
func HealthDB(db *orm.DB) router.HandlerFunc {
	return func(ctx router.Context) {
		ctx.SetHeader("Content-Type", "application/json")

		if db == nil || db.RawConn() == nil {
			ctx.WriteStatus(503)
			_, _ = ctx.Write([]byte(bodyHealthFail))
			return
		}
		qb := db.Query(&Project{}).Limit(1)
		if _, err := ReadAllProject(qb); err != nil {
			ctx.WriteStatus(503)
			_, _ = ctx.Write([]byte(bodyHealthFail))
			return
		}
		ctx.WriteStatus(200)
		_, _ = ctx.Write([]byte(bodyHealthOK))
	}
}

// TokenRequest es el cuerpo de POST /api/token.
type TokenRequest struct {
	SubjectToken string
}

func (t *TokenRequest) IsNil() bool { return t == nil }
func (t *TokenRequest) EncodeFields(w model.FieldWriter) {
	w.String("subject_token", t.SubjectToken)
}
func (t *TokenRequest) DecodeFields(r model.FieldReader) {
	t.SubjectToken, _ = r.String("subject_token")
}

// TokenResponse es la respuesta de POST /api/token.
type TokenResponse struct {
	Token  string
	Email  string
	Name   string
	Avatar string
}

func (t *TokenResponse) IsNil() bool { return t == nil }
func (t *TokenResponse) EncodeFields(w model.FieldWriter) {
	w.String("token", t.Token)
	w.String("email", t.Email)
	w.String("name", t.Name)
	w.String("avatar", t.Avatar)
}
func (t *TokenResponse) DecodeFields(r model.FieldReader) {
	t.Token, _ = r.String("token")
	t.Email, _ = r.String("email")
	t.Name, _ = r.String("name")
	t.Avatar, _ = r.String("avatar")
}

// Token emite un token de autorización project-scoped mediante token exchange (RFC 8693).
func Token(db *orm.DB, authMod *authority.Module, rbacSvc *rbac.Service, secret []byte, ids model.IDGenerator) router.HandlerFunc {
	return func(ctx router.Context) {
		ctx.SetHeader("Content-Type", "application/json")
		projectID := ctx.UserID()
		if projectID == "" {
			ctx.WriteStatus(401)
			return
		}

		var body TokenRequest
		if err := ctx.Decode(&body); err != nil || body.SubjectToken == "" {
			ctx.WriteStatus(400)
			return
		}

		sess, err := authMod.GetSession(body.SubjectToken)
		if err != nil {
			if aerr := RecordAudit(db, ids, projectID, AuditTokenDenied, projectID, ""); aerr != nil {
				fmt.Println("audit:", aerr)
			}
			ctx.WriteStatus(401)
			return
		}

		token, err := IssueAuthToken(rbacSvc, secret, projectID, sess.UserId)
		if err != nil {
			ctx.WriteStatus(500)
			return
		}
		u, err := authMod.GetUser(sess.UserId)
		if err != nil {
			ctx.WriteStatus(500)
			return
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&TokenResponse{Token: token, Email: u.Email, Name: u.Name, Avatar: u.Avatar})
	}
}

// ResolveUserRequest es el cuerpo de POST /api/users/resolve.
type ResolveUserRequest struct {
	Email string
	Name  string
}

func (r *ResolveUserRequest) IsNil() bool { return r == nil }
func (r *ResolveUserRequest) EncodeFields(w model.FieldWriter) {
	w.String("email", r.Email)
	w.String("name", r.Name)
}
func (r *ResolveUserRequest) DecodeFields(fr model.FieldReader) {
	r.Email, _ = fr.String("email")
	r.Name, _ = fr.String("name")
}

// ResolveUserResponse es la respuesta de POST /api/users/resolve.
type ResolveUserResponse struct {
	Sub string
}

func (r *ResolveUserResponse) IsNil() bool { return r == nil }
func (r *ResolveUserResponse) EncodeFields(w model.FieldWriter) {
	w.String("sub", r.Sub)
}
func (r *ResolveUserResponse) DecodeFields(fr model.FieldReader) {
	r.Sub, _ = fr.String("sub")
}

// ResolveUser busca un usuario por email dentro de la identidad global de iam y lo crea si no existe.
func ResolveUser(db *orm.DB, authMod *authority.Module, ids model.IDGenerator) router.HandlerFunc {
	return func(ctx router.Context) {
		ctx.SetHeader("Content-Type", "application/json")
		projectID := ctx.UserID()
		if projectID == "" {
			ctx.WriteStatus(401)
			return
		}

		var body ResolveUserRequest
		if err := ctx.Decode(&body); err != nil || body.Email == "" {
			ctx.WriteStatus(400)
			return
		}

		u, err := authMod.UserByEmail(body.Email)
		if err != nil {
			u, err = authMod.CreateUser(body.Email, body.Name, "")
			if err != nil {
				ctx.WriteStatus(500)
				return
			}
			if aerr := RecordAudit(db, ids, projectID, AuditIdentityCreatedByClient, body.Email, ""); aerr != nil {
				fmt.Println("audit:", aerr)
			}
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&ResolveUserResponse{Sub: u.Id})
	}
}

// AssignRoleRequest es el cuerpo de POST /api/roles/assign.
type AssignRoleRequest struct {
	UserID   string
	RoleCode string
}

func (r *AssignRoleRequest) IsNil() bool { return r == nil }
func (r *AssignRoleRequest) EncodeFields(w model.FieldWriter) {
	w.String("user_id", r.UserID)
	w.String("role_code", r.RoleCode)
}
func (r *AssignRoleRequest) DecodeFields(fr model.FieldReader) {
	r.UserID, _ = fr.String("user_id")
	r.RoleCode, _ = fr.String("role_code")
}

type AssignRoleResponse struct{}

func (r *AssignRoleResponse) IsNil() bool                       { return r == nil }
func (r *AssignRoleResponse) EncodeFields(w model.FieldWriter)  {}
func (r *AssignRoleResponse) DecodeFields(fr model.FieldReader) {}

// AssignRole concede roleCode al usuario en el proyecto del llamante.
func AssignRole(db *orm.DB, rbacSvc *rbac.Service, ids model.IDGenerator) router.HandlerFunc {
	return func(ctx router.Context) {
		ctx.SetHeader("Content-Type", "application/json")
		projectID := ctx.UserID()
		if projectID == "" {
			ctx.WriteStatus(401)
			return
		}

		var body AssignRoleRequest
		if err := ctx.Decode(&body); err != nil {
			ctx.WriteStatus(400)
			return
		}
		if body.UserID == "" || body.RoleCode == "" {
			ctx.WriteStatus(400)
			return
		}

		if err := rbacSvc.AssignRoleByCode(projectID, body.UserID, model.RoleCode(body.RoleCode)); err != nil {
			if rbac.IsRoleNotFound(err) {
				ctx.WriteStatus(404)
				return
			}
			ctx.WriteStatus(500)
			return
		}
		if aerr := RecordAudit(db, ids, projectID, AuditRoleGrantedByClient, body.UserID+":"+body.RoleCode, ""); aerr != nil {
			fmt.Println("audit:", aerr)
		}
		ctx.WriteStatus(200)
		_ = ctx.Encode(&AssignRoleResponse{})
	}
}
