//go:build !wasm

package tests

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"webtyp.com/auth/authority"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/rbac"
	"webtyp.com/router"
	"webtyp.com/router/mock"
	"webtyp.com/sqlt"
	"webtyp.com/storage/mem"
	"github.com/veltylabs/iam/config"
)

// testPanelOrigin es el origen del panel en tests — lo que NewProductionBackend
// exige vía IAM_PANEL_ORIGIN.
const testPanelOrigin = "https://iam.velty.cl"

func migrateTestDB(t *testing.T, db *orm.DB) {
	t.Helper()
	// mem.New() auto-vivifies tables on insert; only migrate real SQL connections.
	if strings.Contains(fmt.Sprintf("%T", db.RawConn()), "mem") {
		return
	}
	if err := config.MigrateAll(db.RawConn(), sqlt.NewCompiler()); err != nil {
		t.Fatal(err)
	}
}

// setup arma el motor de producción (Google mock vía variables de entorno
// de prueba) contra una base en memoria.
func setup(t *testing.T) (*orm.DB, *authority.Module, *rbac.Service, model.IDGenerator) {
	t.Helper()
	t.Setenv(config.EnvGoogleClientID, "test-client-id")
	t.Setenv(config.EnvGoogleSecret, "test-client-secret")
	t.Setenv(config.EnvGoogleRedirectURL, "http://localhost:8080/oauth/callback/google")
	t.Setenv(config.EnvJWTSecret, "test-jwt-secret-32-bytes-long-0000")
	if os.Getenv(config.EnvPanelOrigin) == "" {
		t.Setenv(config.EnvPanelOrigin, testPanelOrigin)
	}

	db := orm.New(mem.New())
	migrateTestDB(t, db)
	ids, err := config.NewIDs()
	if err != nil {
		t.Fatalf("ids: %v", err)
	}
	authMod, rbacSvc, err := config.NewProductionAuth(db, ids)
	if err != nil {
		t.Fatalf("NewProductionAuth: %v", err)
	}
	return db, authMod, rbacSvc, ids
}

// setupBackend arma el Backend completo (incluye JWTSecret) para los tests
// de la Etapa 3/4 (routes.Token, CORS) que necesitan firmar/verificar.
func setupBackend(t *testing.T) *config.Backend {
	t.Helper()
	t.Setenv(config.EnvGoogleClientID, "test-client-id")
	t.Setenv(config.EnvGoogleSecret, "test-client-secret")
	t.Setenv(config.EnvGoogleRedirectURL, "http://localhost:8080/oauth/callback/google")
	t.Setenv(config.EnvJWTSecret, "test-jwt-secret-32-bytes-long-0000")
	if os.Getenv(config.EnvPanelOrigin) == "" {
		t.Setenv(config.EnvPanelOrigin, testPanelOrigin)
	}

	db := orm.New(mem.New())
	migrateTestDB(t, db)
	ids, err := config.NewIDs()
	if err != nil {
		t.Fatalf("ids: %v", err)
	}
	backend, err := config.NewProductionBackend(db, ids)
	if err != nil {
		t.Fatalf("NewProductionBackend: %v", err)
	}
	return backend
}

func setupPanel(t *testing.T, adminEmail string) (*config.Backend, *mock.Router) {
	t.Helper()
	return setupPanelWithOrigin(t, adminEmail, testPanelOrigin)
}

// setupPanelWithOrigin arma el panel fijando IAM_PANEL_ORIGIN — lo necesitan
// todos los tests de la guarda de origen.
func setupPanelWithOrigin(t *testing.T, adminEmail, origin string) (*config.Backend, *mock.Router) {
	t.Helper()
	b := setupBackend(t)
	b.Origin = origin
	r := &mock.Router{}
	r.Configure(mock.Config{
		Authn:     b.Auth.Authenticate(),
		Authorize: config.Authorizer(b.RBAC),
	})
	if err := config.EnsurePolicy(b.RBAC, b.Auth, []string{adminEmail}); err != nil {
		t.Fatalf("EnsurePolicy: %v", err)
	}
	if u, err := b.Auth.UserByEmail(adminEmail); err == nil {
		_ = b.RBAC.AssignRole(config.IAMProjectID, u.Id, config.RoleIAMClientID)
	}
	config.RegisterRoutes(r, b, []string{adminEmail})
	return b, r
}

func issueSSO(t *testing.T, b *config.Backend, userID string) router.Cookie {
	t.Helper()
	ctx := &mock.Context{}
	if err := b.Auth.IssueSession(ctx, userID); err != nil {
		t.Fatalf("IssueSession: %v", err)
	}
	c, ok := ctx.Cookie(config.CookieSession)
	if !ok {
		t.Fatalf("cookie %s not set on context", config.CookieSession)
	}
	return c
}

func issueStepUp(t *testing.T, b *config.Backend, userID string) router.Cookie {
	t.Helper()
	ctx := &mock.Context{}
	if err := b.Auth.IssueStepUpSession(ctx, userID); err != nil {
		t.Fatalf("IssueStepUpSession: %v", err)
	}
	c, ok := ctx.Cookie("__Host-" + config.CookieAdmin)
	if !ok {
		c, ok = ctx.Cookie(config.CookieAdmin)
	}
	if !ok {
		t.Fatalf("cookie %s not set on context", config.CookieAdmin)
	}
	return c
}

// adminCtx devuelve un mock.Context con la sesión ya emitida para email y
// con Sec-Fetch-Site: same-origin puesto. Es el "camino feliz" que casi
// todos los tests nuevos necesitan.
func adminCtx(t *testing.T, b *config.Backend, email string) *mock.Context {
	t.Helper()
	u, err := b.Auth.UserByEmail(email)
	if err != nil {
		u, err = b.Auth.CreateUser(email, email, "")
		if err != nil {
			t.Fatalf("adminCtx CreateUser: %v", err)
		}
	}
	ctx := &mock.Context{}
	c := issueStepUp(t, b, u.Id)
	ctx.SetCookie(c)
	ctx.SetUserID(u.Id)
	ctx.SetAssurance(model.AssuranceMultiFactor)
	ctx.SetHeader("Sec-Fetch-Site", "same-origin")
	return ctx
}

// crossSiteCtx es el atacante hermano: misma sesión válida, pero la petición
// viene de otro subdominio (Sec-Fetch-Site: cross-site, Origin de misitio).
func crossSiteCtx(t *testing.T, b *config.Backend, email string) *mock.Context {
	t.Helper()
	u, err := b.Auth.UserByEmail(email)
	if err != nil {
		u, err = b.Auth.CreateUser(email, email, "")
		if err != nil {
			t.Fatalf("crossSiteCtx CreateUser: %v", err)
		}
	}
	ctx := &mock.Context{}
	c := issueStepUp(t, b, u.Id)
	ctx.SetCookie(c)
	ctx.SetUserID(u.Id)
	ctx.SetAssurance(model.AssuranceMultiFactor)
	ctx.SetHeader("Sec-Fetch-Site", "cross-site")
	ctx.SetHeader("Origin", "https://misitio.velty.cl")
	return ctx
}

// lastAudit devuelve la última fila de auditoría, para afirmar sobre ella
// sin repetir la consulta.
func lastAudit(t *testing.T, db *orm.DB) config.AuditEntry {
	t.Helper()
	qb := db.Query(&config.AuditEntry{})
	entries, err := config.ReadAllAuditEntry(qb)
	if err != nil {
		t.Fatalf("lastAudit: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("lastAudit: no audit entries")
	}
	return *entries[len(entries)-1]
}

// setupLocal arma el motor de desarrollo local, sin variables GOOGLE_*.
func setupLocal(t *testing.T) (*orm.DB, *authority.Module, *rbac.Service, model.IDGenerator) {
	t.Helper()
	db := orm.New(mem.New())
	migrateTestDB(t, db)
	ids, err := config.NewIDs()
	if err != nil {
		t.Fatalf("ids: %v", err)
	}
	authMod, rbacSvc, err := config.NewLocalAuth(db, ids)
	if err != nil {
		t.Fatalf("NewLocalAuth: %v", err)
	}
	return db, authMod, rbacSvc, ids
}
