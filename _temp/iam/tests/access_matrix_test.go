//go:build !wasm

package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"webtyp.com/base64"
	"webtyp.com/crypto/asym"
	"webtyp.com/json"
	tinyjwt "webtyp.com/jwt"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/router/mock"
	"webtyp.com/router/routertest"
	"github.com/veltylabs/iam/client"
	"github.com/veltylabs/iam/config"
)

var adminRoutes = []struct {
	method string
	path   string
}{
	{"GET", config.PathAdminProjects},
	{"POST", config.PathAdminProjects},
	{"POST", config.PathAdminProjectKey},
	{"POST", config.PathAdminProjectActive},
	{"GET", config.PathAdminRoles},
	{"POST", config.PathAdminRoles},
	{"POST", config.PathAdminRoleTTL},
	{"POST", config.PathAdminRoleDelete},
	{"GET", config.PathAdminRoleUsers},
	{"POST", config.PathAdminUserAssign},
	{"POST", config.PathAdminUserRevoke},
	{"GET", config.PathAdminAudit},
	{"GET", router.IntrospectionPath},
	{"GET", config.PathHealthDB},
}

func setupAccessMatrix(t *testing.T) (*config.Backend, *mock.Router) {
	t.Helper()
	b := setupBackend(t)
	r := &mock.Router{}
	r.Configure(mock.Config{
		Authn:     b.Auth.Authenticate(),
		Authorize: config.Authorizer(b.RBAC),
	})
	if err := config.EnsurePolicy(b.RBAC, b.Auth, []string{"admin@example.com"}); err != nil {
		t.Fatalf("EnsurePolicy: %v", err)
	}
	config.RegisterRoutes(r, b, []string{"admin@example.com"})
	return b, r
}

// T1: every /api/admin/* except me, plus /_routes and /api/health/db: anonymous -> 403
func TestT1_AnonymousAdminRoutes403(t *testing.T) {
	_, r := setupAccessMatrix(t)
	for _, route := range adminRoutes {
		ctx := &mock.Context{InMethod: route.method, InPath: route.path}
		r.Invoke(route.method, route.path, ctx)
		if ctx.Status != 403 {
			t.Errorf("T1 anonymous %s %s: got %d, want 403", route.method, route.path, ctx.Status)
		}
	}
}

// T2: same routes with an SSO session only -> 401, header router.ChallengeMFARequired
func TestT2_SSOOnlyMFARequired401(t *testing.T) {
	b, r := setupAccessMatrix(t)
	adminU, err := b.Auth.UserByEmail("admin@example.com")
	if err != nil {
		adminU, _ = b.Auth.CreateUser("admin@example.com", "Admin", "")
		_ = b.RBAC.AssignRole(config.IAMProjectID, adminU.Id, config.RoleIAMAdminID)
	}
	c := issueSSO(t, b, adminU.Id)

	for _, route := range adminRoutes {
		ctx := &mock.Context{InMethod: route.method, InPath: route.path}
		ctx.SetCookie(c)
		r.Invoke(route.method, route.path, ctx)
		if ctx.Status != 401 {
			t.Errorf("T2 SSO-only %s %s: got %d, want 401", route.method, route.path, ctx.Status)
		}
		authHdr := ctx.GetHeader(router.HeaderWWWAuthenticate)
		if authHdr != router.ChallengeMFARequired {
			t.Errorf("T2 header %s %s: got %q, want %q", route.method, route.path, authHdr, router.ChallengeMFARequired)
		}
	}
}

// T3: step-up session of a non-admin -> 403
func TestT3_NonAdminStepUp403(t *testing.T) {
	b, r := setupAccessMatrix(t)
	nonAdmin, err := b.Auth.CreateUser("user@example.com", "User", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	c := issueStepUp(t, b, nonAdmin.Id)

	for _, route := range adminRoutes {
		ctx := &mock.Context{InMethod: route.method, InPath: route.path}
		ctx.SetCookie(c)
		r.Invoke(route.method, route.path, ctx)
		if ctx.Status != 403 {
			t.Errorf("T3 non-admin %s %s: got %d, want 403", route.method, route.path, ctx.Status)
		}
	}
}

// T4: step-up session of an admin -> 200 on each GET route
func TestT4_AdminStepUp200(t *testing.T) {
	b, r := setupAccessMatrix(t)
	adminU, err := b.Auth.UserByEmail("admin@example.com")
	if err != nil {
		adminU, _ = b.Auth.CreateUser("admin@example.com", "Admin", "")
		_ = b.RBAC.AssignRole(config.IAMProjectID, adminU.Id, config.RoleIAMAdminID)
	}
	c := issueStepUp(t, b, adminU.Id)

	_ = config.CreateProject(b.DB, "p1", "Project 1", "")
	_ = b.RBAC.CreateRole("p1", "role-1", "r1", "Role 1", "")

	type routeItem struct {
		path   string
		inPath string
	}
	getRoutes := []routeItem{
		{path: config.PathAdminProjects, inPath: config.PathAdminProjects},
		{path: config.PathAdminRoles, inPath: config.PathAdminRoles + "?project_id=p1"},
		{path: config.PathAdminRoleUsers, inPath: config.PathAdminRoleUsers + "?project_id=p1&code=r1"},
		{path: config.PathAdminAudit, inPath: config.PathAdminAudit},
		{path: router.IntrospectionPath, inPath: router.IntrospectionPath},
		{path: config.PathHealthDB, inPath: config.PathHealthDB},
	}

	for _, item := range getRoutes {
		ctx := &mock.Context{InMethod: "GET", InPath: item.inPath}
		ctx.SetCookie(c)
		r.Invoke("GET", item.path, ctx)
		if ctx.Status != 200 {
			t.Errorf("T4 admin GET %s: got %d, want 200", item.path, ctx.Status)
		}
	}
}

// T5: /api/token with no bearer; with an assertion for another URL (htu); replayed assertion
func TestT5_MachineAssertionValidation(t *testing.T) {
	b, r := setupAccessMatrix(t)
	pub, priv, err := asym.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	projectID := "proj-t5"
	_ = config.CreateProject(b.DB, projectID, "Proj T5", base64.URLEncode(pub))
	_ = b.RBAC.AssignRole(config.IAMProjectID, projectID, config.RoleIAMClientID)

	// 1. No bearer
	ctxNoBearer := &mock.Context{InMethod: "POST", InPath: config.PathToken}
	r.Invoke("POST", config.PathToken, ctxNoBearer)
	if ctxNoBearer.Status != 403 {
		t.Errorf("no bearer: got %d, want 403", ctxNoBearer.Status)
	}

	// 2. Assertion for wrong URL (htu)
	wrongURL := b.Origin + "/api/wrong"
	aWrong, _ := tinyjwt.NewClientAssertion(projectID, b.Origin, "POST", wrongURL)
	sigWrong, _ := tinyjwt.SignClientAssertion(priv, aWrong)
	ctxWrong := &mock.Context{InMethod: "POST", InPath: config.PathToken}
	ctxWrong.SetHeader("Authorization", "Bearer "+sigWrong)
	r.Invoke("POST", config.PathToken, ctxWrong)
	if ctxWrong.Status != 403 {
		t.Errorf("wrong htu: got %d, want 403", ctxWrong.Status)
	}

	// 3. Replay assertion (same JTI twice)
	correctURL := b.Origin + config.PathToken
	aValid, _ := tinyjwt.NewClientAssertion(projectID, b.Origin, "POST", correctURL)
	sigValid, _ := tinyjwt.SignClientAssertion(priv, aValid)

	// First call -> reaches handler (token request needs body, so handler returns 400 or 401, not 403 from gate)
	ctxFirst := &mock.Context{InMethod: "POST", InPath: config.PathToken}
	ctxFirst.SetHeader("Authorization", "Bearer "+sigValid)
	r.Invoke("POST", config.PathToken, ctxFirst)
	if ctxFirst.Status == 403 {
		t.Errorf("first assertion call: got 403, want authorized (got status %d)", ctxFirst.Status)
	}

	// Second call with identical token -> replayed assertion -> gate 403
	ctxSecond := &mock.Context{InMethod: "POST", InPath: config.PathToken}
	ctxSecond.SetHeader("Authorization", "Bearer "+sigValid)
	r.Invoke("POST", config.PathToken, ctxSecond)
	if ctxSecond.Status != 403 {
		t.Errorf("replayed assertion call: got %d, want 403", ctxSecond.Status)
	}
}

// T6: assertion signed by a project marked inactive; by a project with no key
func TestT6_MachineInactiveOrNoKey(t *testing.T) {
	b, r := setupAccessMatrix(t)
	pub, priv, err := asym.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	// Inactive project
	projInactive := "proj-inactive"
	_ = config.CreateProject(b.DB, projInactive, "Inactive", base64.URLEncode(pub))
	_ = config.SetProjectActive(b.DB, projInactive, false)
	_ = b.RBAC.AssignRole(config.IAMProjectID, projInactive, config.RoleIAMClientID)

	aInactive, _ := tinyjwt.NewClientAssertion(projInactive, b.Origin, "POST", b.Origin+config.PathToken)
	sigInactive, _ := tinyjwt.SignClientAssertion(priv, aInactive)
	ctxInactive := &mock.Context{InMethod: "POST", InPath: config.PathToken}
	ctxInactive.SetHeader("Authorization", "Bearer "+sigInactive)
	r.Invoke("POST", config.PathToken, ctxInactive)
	if ctxInactive.Status != 403 {
		t.Errorf("inactive project assertion: got %d, want 403", ctxInactive.Status)
	}

	// Project with no key
	projNoKey := "proj-nokey"
	_ = config.CreateProject(b.DB, projNoKey, "No Key", "")
	_ = b.RBAC.AssignRole(config.IAMProjectID, projNoKey, config.RoleIAMClientID)

	aNoKey, _ := tinyjwt.NewClientAssertion(projNoKey, b.Origin, "POST", b.Origin+config.PathToken)
	sigNoKey, _ := tinyjwt.SignClientAssertion(priv, aNoKey)
	ctxNoKey := &mock.Context{InMethod: "POST", InPath: config.PathToken}
	ctxNoKey.SetHeader("Authorization", "Bearer "+sigNoKey)
	r.Invoke("POST", config.PathToken, ctxNoKey)
	if ctxNoKey.Status != 403 {
		t.Errorf("project with no key assertion: got %d, want 403", ctxNoKey.Status)
	}
}

// T7: AssignRole assigns in the calling project only; ResolveUser works; Token exchanges SSO session id
func TestT7_MachineOperations(t *testing.T) {
	b, r := setupAccessMatrix(t)
	pub, priv, err := asym.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	projectID := "proj-t7"
	_ = config.CreateProject(b.DB, projectID, "Proj T7", base64.URLEncode(pub))
	_ = b.RBAC.AssignRole(config.IAMProjectID, projectID, config.RoleIAMClientID)
	_ = b.RBAC.CreateRole(projectID, "r1", "editor", "Editor", "")

	sign := func(method, path string) string {
		a, _ := tinyjwt.NewClientAssertion(projectID, b.Origin, method, b.Origin+path)
		s, _ := tinyjwt.SignClientAssertion(priv, a)
		return s
	}

	// 1. ResolveUser
	ctxRes := &mock.Context{InMethod: "POST", InPath: config.PathUsersResolve}
	ctxRes.SetHeader("Authorization", "Bearer "+sign("POST", config.PathUsersResolve))
	var resBody []byte
	_ = json.Encode(&config.ResolveUserRequest{Email: "user@t7.com", Name: "T7 User"}, &resBody)
	ctxRes.InBody = resBody
	r.Invoke("POST", config.PathUsersResolve, ctxRes)
	if ctxRes.Status != 200 {
		t.Fatalf("ResolveUser: got %d, want 200", ctxRes.Status)
	}
	var resResp config.ResolveUserResponse
	_ = json.Decode(ctxRes.ResponseBody(), &resResp)
	if resResp.Sub == "" {
		t.Fatal("ResolveUser returned empty sub")
	}

	// 2. AssignRole in calling project
	ctxAssign := &mock.Context{InMethod: "POST", InPath: config.PathRolesAssign}
	ctxAssign.SetHeader("Authorization", "Bearer "+sign("POST", config.PathRolesAssign))
	var assignBody []byte
	_ = json.Encode(&config.AssignRoleRequest{UserID: resResp.Sub, RoleCode: "editor"}, &assignBody)
	ctxAssign.InBody = assignBody
	r.Invoke("POST", config.PathRolesAssign, ctxAssign)
	if ctxAssign.Status != 200 {
		t.Fatalf("AssignRole: got %d, want 200", ctxAssign.Status)
	}
	roles, err := b.RBAC.GetUserRoles(projectID, resResp.Sub)
	if err != nil || len(roles) != 1 || roles[0].Code != "editor" {
		t.Fatalf("roles in project: %+v, err=%v", roles, err)
	}

	// 3. Token exchange with valid SSO session
	ssoC := issueSSO(t, b, resResp.Sub)
	ctxTok := &mock.Context{InMethod: "POST", InPath: config.PathToken}
	ctxTok.SetHeader("Authorization", "Bearer "+sign("POST", config.PathToken))
	var tokBody []byte
	_ = json.Encode(&config.TokenRequest{SubjectToken: ssoC.Value}, &tokBody)
	ctxTok.InBody = tokBody
	r.Invoke("POST", config.PathToken, ctxTok)
	if ctxTok.Status != 200 {
		t.Fatalf("Token exchange valid: got %d, want 200", ctxTok.Status)
	}

	// Invalid SSO session -> 401
	ctxBadTok := &mock.Context{InMethod: "POST", InPath: config.PathToken}
	ctxBadTok.SetHeader("Authorization", "Bearer "+sign("POST", config.PathToken))
	var badTokBody []byte
	_ = json.Encode(&config.TokenRequest{SubjectToken: "invalid-session-id"}, &badTokBody)
	ctxBadTok.InBody = badTokBody
	r.Invoke("POST", config.PathToken, ctxBadTok)
	if ctxBadTok.Status != 401 {
		t.Fatalf("Token exchange invalid: got %d, want 401", ctxBadTok.Status)
	}
}

// Inventory: AssertPublicRoutes
func TestInventory_PublicRoutes(t *testing.T) {
	_, r := setupAccessMatrix(t)
	routertest.AssertPublicRoutes(t, r,
		"GET "+config.PathHealth,
		"POST /logout",
		"GET /oauth/google",
		"GET /oauth/callback/google",
	)
}

// /api/admin/me
func TestAdminMe(t *testing.T) {
	b, r := setupAccessMatrix(t)
	adminU, err := b.Auth.UserByEmail("admin@example.com")
	if err != nil {
		adminU, _ = b.Auth.CreateUser("admin@example.com", "Admin", "")
		_ = b.RBAC.AssignRole(config.IAMProjectID, adminU.Id, config.RoleIAMAdminID)
	}
	nonAdminU, _ := b.Auth.CreateUser("user@example.com", "User", "")

	// 1. Anonymous -> 403
	ctxAnon := &mock.Context{InMethod: "GET", InPath: config.PathAdminMe}
	r.Invoke("GET", config.PathAdminMe, ctxAnon)
	if ctxAnon.Status != 403 {
		t.Errorf("me anonymous: got %d, want 403", ctxAnon.Status)
	}

	// 2. SSO admin -> 200, IsAdmin=true, MultiFactor=false
	ssoAdmin := issueSSO(t, b, adminU.Id)
	ctxSSO := &mock.Context{InMethod: "GET", InPath: config.PathAdminMe}
	ctxSSO.SetCookie(ssoAdmin)
	r.Invoke("GET", config.PathAdminMe, ctxSSO)
	if ctxSSO.Status != 200 {
		t.Fatalf("me SSO: got %d, want 200", ctxSSO.Status)
	}
	var meSSO config.AdminMeResponse
	_ = json.Decode(ctxSSO.ResponseBody(), &meSSO)
	if !meSSO.IsAdmin || meSSO.MultiFactor {
		t.Errorf("me SSO admin: got IsAdmin=%v MultiFactor=%v, want true/false", meSSO.IsAdmin, meSSO.MultiFactor)
	}

	// 3. SSO non-admin -> 200, IsAdmin=false
	ssoNonAdmin := issueSSO(t, b, nonAdminU.Id)
	ctxNonAdmin := &mock.Context{InMethod: "GET", InPath: config.PathAdminMe}
	ctxNonAdmin.SetCookie(ssoNonAdmin)
	r.Invoke("GET", config.PathAdminMe, ctxNonAdmin)
	if ctxNonAdmin.Status != 200 {
		t.Fatalf("me non-admin: got %d, want 200", ctxNonAdmin.Status)
	}
	var meNonAdmin config.AdminMeResponse
	_ = json.Decode(ctxNonAdmin.ResponseBody(), &meNonAdmin)
	if meNonAdmin.IsAdmin {
		t.Errorf("me non-admin: got IsAdmin=true, want false")
	}

	// 4. Step-up admin -> 200, MultiFactor=true
	stepUpAdmin := issueStepUp(t, b, adminU.Id)
	ctxStepUp := &mock.Context{InMethod: "GET", InPath: config.PathAdminMe}
	ctxStepUp.SetCookie(stepUpAdmin)
	r.Invoke("GET", config.PathAdminMe, ctxStepUp)
	if ctxStepUp.Status != 200 {
		t.Fatalf("me step-up: got %d, want 200", ctxStepUp.Status)
	}
	var meStepUp config.AdminMeResponse
	_ = json.Decode(ctxStepUp.ResponseBody(), &meStepUp)
	if !meStepUp.MultiFactor {
		t.Errorf("me step-up admin: got MultiFactor=false, want true")
	}
}

// EnsurePolicy: idempotent
func TestEnsurePolicyIdempotent(t *testing.T) {
	b := setupBackend(t)
	adminEmail := "admin@example.com"
	u, _ := b.Auth.CreateUser(adminEmail, "Admin", "")

	err1 := config.EnsurePolicy(b.RBAC, b.Auth, []string{adminEmail})
	if err1 != nil {
		t.Fatalf("first EnsurePolicy: %v", err1)
	}
	roles1, _ := b.RBAC.GetUserRoles(config.IAMProjectID, u.Id)

	err2 := config.EnsurePolicy(b.RBAC, b.Auth, []string{adminEmail})
	if err2 != nil {
		t.Fatalf("second EnsurePolicy: %v", err2)
	}
	roles2, _ := b.RBAC.GetUserRoles(config.IAMProjectID, u.Id)

	if len(roles1) != len(roles2) || len(roles1) == 0 {
		t.Errorf("roles count mismatch or empty: %d vs %d", len(roles1), len(roles2))
	}
	if !b.RBAC.Can(config.IAMProjectID, u.Id, config.ResourceProject, model.Read) {
		t.Error("admin user does not have permission after EnsurePolicy")
	}
}

// originHost
func TestOriginHost(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://iam.velty.cl", "iam.velty.cl"},
		{"https://localhost:8080", "localhost"},
		{"http://example.com:3000/path", "example.com"},
		{"iam.velty.cl", "iam.velty.cl"},
	}
	for _, c := range cases {
		got := config.OriginHost(c.in)
		if got != c.want {
			t.Errorf("OriginHost(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// SetProjectKey: garbage key rejected; valid SPKI accepted
func TestSetProjectKeyValidation(t *testing.T) {
	b := setupBackend(t)
	const proj = "proj-keys"
	_ = config.CreateProject(b.DB, proj, "Proj Keys", "")

	// Garbage key
	errGarbage := config.SetProjectKey(b.DB, proj, "not-a-valid-key")
	if errGarbage == nil {
		t.Error("expected error for garbage public key, got nil")
	}

	// Valid SPKI
	pub, _, err := asym.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	errValid := config.SetProjectKey(b.DB, proj, base64.URLEncode(pub))
	if errValid != nil {
		t.Errorf("unexpected error for valid SPKI: %v", errValid)
	}
}

// Client consumer end-to-end against httptest
func TestConsumerClientEndToEnd(t *testing.T) {
	var handler http.HandlerFunc
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if handler != nil {
			handler(w, req)
		}
	}))
	defer srv.Close()

	t.Setenv(config.EnvPanelOrigin, srv.URL)
	b, r := setupAccessMatrix(t)

	pub, priv, err := asym.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	projectID := "proj-consumer-e2e"
	_ = config.CreateProject(b.DB, projectID, "Consumer E2E", base64.URLEncode(pub))
	_ = b.RBAC.AssignRole(config.IAMProjectID, projectID, config.RoleIAMClientID)
	_ = b.RBAC.CreateRole(projectID, "r1", "editor", "Editor", "")

	u, err := b.Auth.CreateUser("consumeruser@example.com", "Consumer User", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	ssoC := issueSSO(t, b, u.Id)

	// Wrap mock router in standard http.Handler for httptest
	handler = func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		ctx := &mock.Context{
			InMethod: req.Method,
			InPath:   req.URL.Path,
			InBody:   body,
		}
		for k, v := range req.Header {
			if len(v) > 0 {
				ctx.SetHeader(k, v[0])
			}
		}
		for _, c := range req.Cookies() {
			ctx.SetCookie(router.Cookie{Name: c.Name, Value: c.Value})
		}
		r.Invoke(req.Method, req.URL.Path, ctx)

		w.WriteHeader(ctx.Status)
		for _, h := range []string{"Content-Type", router.HeaderWWWAuthenticate} {
			if v := ctx.GetHeader(h); v != "" {
				w.Header().Set(h, v)
			}
		}
		w.Write(ctx.ResponseBody())
	}

	c, err := client.New(client.Config{
		BaseURL:    srv.URL,
		ProjectID:  projectID,
		PrivateKey: priv,
	})
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	// 1. ResolveUser
	sub, err := c.ResolveUser("newe2e@example.com", "New E2E")
	if err != nil {
		t.Fatalf("ResolveUser: %v", err)
	}
	if sub == "" {
		t.Fatal("ResolveUser returned empty sub")
	}

	// 2. AssignRole
	if err := c.AssignRole(sub, "editor"); err != nil {
		t.Fatalf("AssignRole: %v", err)
	}

	// 3. Authn middleware with SSO cookie
	var authnUserID string
	next := func(ctx router.Context) {
		authnUserID = ctx.UserID()
	}
	mCtx := &mock.Context{}
	mCtx.SetCookie(router.Cookie{Name: client.SSOCookieName, Value: ssoC.Value})
	c.Authn()(next)(mCtx)

	if authnUserID != u.Id {
		t.Errorf("Authn userID: got %q, want %q", authnUserID, u.Id)
	}
}

// CSRF defense in depth: step-up admin POST with Sec-Fetch-Site: same-site and a sibling Origin -> 403
func TestCSRFDefenseInDepth(t *testing.T) {
	b, r := setupAccessMatrix(t)
	adminU, _ := b.Auth.CreateUser("admin@example.com", "Admin", "")
	c := issueStepUp(t, b, adminU.Id)

	ctx := &mock.Context{InMethod: "POST", InPath: config.PathAdminProjects}
	ctx.SetCookie(c)
	ctx.SetHeader("Sec-Fetch-Site", "same-site")
	ctx.SetHeader("Origin", "https://sibling.velty.cl")
	var body []byte
	_ = json.Encode(&config.AdminCreateProjectRequest{Name: "CSRF Project"}, &body)
	ctx.InBody = body

	r.Invoke("POST", config.PathAdminProjects, ctx)
	if ctx.Status != 403 {
		t.Errorf("CSRF sibling origin: got status %d, want 403", ctx.Status)
	}
}
