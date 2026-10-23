//go:build !wasm

package tests

import (
	"testing"

	"webtyp.com/router/mock"
	"github.com/veltylabs/iam/config"
)

// adminPostPaths son las ocho rutas POST de /api/admin/* que mutan estado.
// Cada una lleva la guarda RequireSameOrigin como defensa en profundidad.
var adminPostPaths = []string{
	config.PathAdminProjects,
	config.PathAdminProjectKey,
	config.PathAdminProjectActive,
	config.PathAdminRoles,
	config.PathAdminRoleTTL,
	config.PathAdminRoleDelete,
	config.PathAdminUserAssign,
	config.PathAdminUserRevoke,
}

// TestAdminMutationRejectsCrossSite: los ocho POST con el atacante hermano
// (sesión válida, Sec-Fetch-Site: cross-site, Origin de otro subdominio) dan
// 403, y el estado no cambió.
func TestAdminMutationRejectsCrossSite(t *testing.T) {
	b, r := setupPanel(t, "admin@example.com")

	for _, path := range adminPostPaths {
		ctx := crossSiteCtx(t, b, "admin@example.com")
		ctx.InMethod = "POST"
		ctx.InPath = path
		r.Invoke("POST", path, ctx)
		if ctx.Status != 403 {
			t.Errorf("POST %s cross-site: got %d, want 403", path, ctx.Status)
		}
	}

	// El estado no cambió: ningún proyecto se creó por el camino denegado.
	projects, err := config.ListProjects(b.DB)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("cross-site requests mutated state: %d projects", len(projects))
	}
}

// TestAdminMutationRejectsMissingOriginSignals: sin Origin ni Sec-Fetch-Site
// también es 403 — el panel siempre las manda, y lo único que llega sin
// ellas es un cliente que no es el panel.
func TestAdminMutationRejectsMissingOriginSignals(t *testing.T) {
	b, r := setupPanel(t, "admin@example.com")
	u, err := b.Auth.CreateUser("admin@example.com", "Admin", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	for _, path := range adminPostPaths {
		ctx := adminCtx(t, b, "admin@example.com")
		// adminCtx pone Sec-Fetch-Site; acá se quita a propósito.
		ctx.SetHeader("Sec-Fetch-Site", "")
		ctx.SetUserID(u.Id)
		ctx.InMethod = "POST"
		ctx.InPath = path
		r.Invoke("POST", path, ctx)
		if ctx.Status != 403 {
			t.Errorf("POST %s without origin signals: got %d, want 403", path, ctx.Status)
		}
	}
}

// TestAdminMutationAcceptsSameOrigin: el camino feliz sigue funcionando —
// crear un proyecto desde el propio origen devuelve 201.
func TestAdminMutationAcceptsSameOrigin(t *testing.T) {
	b, r := setupPanel(t, "admin@example.com")

	ctx := adminCtx(t, b, "admin@example.com")
	ctx.InMethod = "POST"
	ctx.InPath = config.PathAdminProjects
	ctx.InBody = encodeBody(t, &config.AdminCreateProjectRequest{Name: "Same Origin App"})
	r.Invoke("POST", config.PathAdminProjects, ctx)
	if ctx.Status != 201 {
		t.Fatalf("POST %s same-origin: got %d, want 201", config.PathAdminProjects, ctx.Status)
	}
}

// TestAdminGetsDoNotRequireOrigin: los GET no mutan y no llevan la guarda —
// funcionan sin ninguna señal de origen.
func TestAdminGetsDoNotRequireOrigin(t *testing.T) {
	b, r := setupPanel(t, "admin@example.com")
	u, err := b.Auth.UserByEmail("admin@example.com")
	if err != nil {
		u, _ = b.Auth.CreateUser("admin@example.com", "Admin", "")
	}
	c := issueStepUp(t, b, u.Id)

	gets := []string{
		config.PathAdminMe,
		config.PathAdminProjects,
		config.PathAdminAudit,
	}
	for _, path := range gets {
		// Sin ninguna cabecera de origen: los GET no la exigen.
		ctx := &mock.Context{InMethod: "GET", InPath: path}
		ctx.SetCookie(c)
		r.Invoke("GET", path, ctx)
		if ctx.Status != 200 {
			t.Errorf("GET %s without origin signals: got %d, want 200", path, ctx.Status)
		}
	}
}
