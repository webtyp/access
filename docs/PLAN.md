---
PLAN: "feat: access UI — roles, role assignments and audit screens with crudview, ui/ package and browser demo"
TAG: v0.0.3
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 13203505619078721704
---

> Read AGENTS.md at the repo root before any change.

## Tools

Install these exact tools first:
```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
```

## Ecosystem search

1. Existing screens: `veltylabs/iam` previously implemented roles, users, and audit screens as raw hand-written DOM and REST handlers. That was app-local code.
2. The modern WebTyp pattern for reusable administration screens:
   - Module root ships `view.go` implementing `view.Itemizer` (`Item() view.Item`) on domain records and `view.Presenter` constructors (`NewRolesView`, `NewAssignmentsView`, `NewAuditView`).
   - `ui/` package exports `ID`, labels, icons (`ui/svg.go` with `//go:build !wasm`), and `Browser(caller, ids, projectID) ([]platformd.UIModule, error)` plus individual `RolesBrowser`, `AssignmentsBrowser`, `AuditBrowser`.
   - `web/browser/main.go` runs an in-browser demo via `orm.New(mem.New())` + `loopback.ActingAs`.

## Design gate

1. **Prior art**: `veltylabs/service_catalog/ui`, `veltylabs/device_manager/ui`, `webtyp.com/layout/crudview`.
2. **Novice-name test**: `RolesBrowser`, `AssignmentsBrowser`, `AuditBrowser`, `Browser` — clear, standard WebTyp UI constructors.
3. **Complexity ledger**: Zero new concepts. Uses standard `crudview.New`, `platformd.NewUIModule`, and `view.NewCallerLister`.
4. **Where it belongs**: `webtyp/access/ui` alongside `access` module root.
5. **What it deletes**: Eliminates hand-written admin screens across consuming applications (`veltylabs/iam`, `veltylabs/mjosefa-cms`).

## APIs used

| Symbol | Signature | Package |
|---|---|---|
| `crudview.New` | `func New(cfg crudview.Config) (*crudview.CrudView, error)` | `webtyp.com/layout/crudview` |
| `crudview.Config` | `struct { ParentID string; Presenter view.Presenter; IDs model.IDGenerator; NewRecord func() model.Model; ... }` | `webtyp.com/layout/crudview` |
| `platformd.NewUIModule` | `func NewUIModule(id string, label string, icon svg.Icon, view dom.Component) platformd.UIModule` | `webtyp.com/layout/platformd` |
| `platformd.UIModule` | `interface { ID() string; Label() string; Icon() svg.Icon; View() dom.Component }` | `webtyp.com/layout/platformd` |
| `view.NewCallerLister` | `func NewCallerLister(caller router.Caller, ops view.Ops, newList func() model.ModelSlice) *callerLister` | `webtyp.com/view` |
| `view.New` | `func New(b view.Lister, record model.Model, opts ...view.Option) view.Presenter` | `webtyp.com/view` |
| `view.WithTitle` | `func WithTitle(title string) view.Option` | `webtyp.com/view` |
| `view.Item` | `struct { ID string; Label string; Description string; ... }` | `webtyp.com/view` |
| `loopback.ActingAs` | `func ActingAs(tenantID, userID string, mods ...router.OperationModule) router.Caller` | `webtyp.com/router/loopback` |

If a signature in the published tag differs from this table, stop and name the symbol in the PR; never adapt it by guessing.

---

## Stage 1 — `view.go` (Presenter and Itemizer implementations)

Create `view.go` in package `access`:

```go
package access

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

const (
	TitleRoles       = "Roles"
	TitleAssignments = "Assignments"
	TitleAudit       = "Audit"
)

// Item projects a RoleRecord as a view.Item (view.Itemizer).
func (r *RoleRecord) Item() view.Item {
	return view.Item{
		ID:          r.Id,
		Label:       r.Name,
		Description: r.Code,
	}
}

// Item projects an Assignment as a view.Item (view.Itemizer).
func (a *Assignment) Item() view.Item {
	desc := a.RoleCode
	if a.Name != "" {
		desc = a.Name + " (" + a.RoleCode + ")"
	}
	return view.Item{
		ID:          a.Id,
		Label:       a.Email,
		Description: desc,
	}
}

// Item projects an AuditEntry as a view.Item (view.Itemizer).
func (e *AuditEntry) Item() view.Item {
	desc := e.Action + " " + e.Target
	if e.Detail != "" {
		desc += " — " + e.Detail
	}
	return view.Item{
		ID:          e.Id,
		Label:       e.ActorEmail,
		Description: desc,
	}
}

// NewRolesView builds the Presenter for the roles CRUD screen.
func NewRolesView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{
			Module: ModelName,
			List:   OpListRoles,
			Save:   OpSaveRoles,
			Delete: OpDeleteRoles,
		},
		func() model.ModelSlice { return &RoleRecordList{} },
	)
	return view.New(b, &RoleRecord{}, view.WithTitle(TitleRoles))
}

// NewAssignmentsView builds the Presenter for the assignments CRUD screen.
func NewAssignmentsView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{
			Module: ModelName,
			List:   OpListAssignments,
			Save:   OpAssignRoles,
			Delete: OpRevokeAssignments,
		},
		func() model.ModelSlice { return &AssignmentList{} },
	)
	return view.New(b, &Assignment{}, view.WithTitle(TitleAssignments))
}

// NewAuditView builds the Presenter for the audit trail view (read-only list).
func NewAuditView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{
			Module: ModelName,
			List:   OpListAudit,
		},
		func() model.ModelSlice { return &AuditEntryList{} },
	)
	return view.New(b, &AuditEntry{}, view.WithTitle(TitleAudit))
}
```

Stage git commit:
`git add -A && git commit -m "feat(access): add view itemizer and presenter constructors"`

---

## Stage 2 — `ui/module.go` & `ui/svg.go`

Create `ui/module.go`:

```go
package ui

import "webtyp.com/svg"

// ID is this module's root identity on client nav rails.
const ID = "access"

// Labels for navigation rail and headers.
const (
	LabelRoles       = "Roles"
	LabelAssignments = "Asignaciones"
	LabelAudit       = "Auditoría"
)

// SVG Icon references.
const (
	IconRoles       = svg.Icon("access_roles")
	IconAssignments = svg.Icon("access_assignments")
	IconAudit       = svg.Icon("access_audit")
)
```

Create `ui/svg.go` with build tag `//go:build !wasm`:

```go
//go:build !wasm

package ui

import "webtyp.com/svg/sprite"

// Icons declares the SVGs for code generation / site extraction.
type Icons struct{}

func (m *Icons) IconSvg() *sprite.Sprite {
	return sprite.NewSprite(
		sprite.Define(IconRoles, "0 0 16 16",
			sprite.Path("M8 1a3 3 0 1 0 0 6 3 3 0 0 0 0-6zM3 13c0-2.5 2.5-4 5-4s5 1.5 5 4v1H3v-1z"),
		),
		sprite.Define(IconAssignments, "0 0 16 16",
			sprite.Path("M11 7V5a3 3 0 1 0-6 0v2H4v7h8V7h-1zm-4-2a1 1 0 1 1 2 0v2H7V5z"),
		),
		sprite.Define(IconAudit, "0 0 16 16",
			sprite.Path("M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zm0 2a5 5 0 1 1 0 10A5 5 0 0 1 8 3zm-.5 2v3.5l3 1.5.5-.8-2.5-1.2V5h-1z"),
		),
	)
}
```

Stage git commit:
`git add -A && git commit -m "feat(ui): add module identity, labels and SVG icons"`

---

## Stage 3 — `ui/browser.go`

Create `ui/browser.go`:

```go
package ui

import (
	"webtyp.com/access"
	"webtyp.com/fmt"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/router"
)

// RolesBrowser builds the roles management UI module.
func RolesBrowser(caller router.Caller, ids model.IDGenerator) (platformd.UIModule, error) {
	v, err := crudview.New(crudview.Config{
		ParentID:  ID + ".roles",
		Presenter: access.NewRolesView(caller),
		IDs:       ids,
		NewRecord: func() model.Model {
			return &access.RoleRecord{}
		},
	})
	if err != nil {
		return nil, err
	}
	return platformd.NewUIModule(ID+".roles", LabelRoles, IconRoles, v), nil
}

// AssignmentsBrowser builds the role assignments UI module.
func AssignmentsBrowser(caller router.Caller, ids model.IDGenerator) (platformd.UIModule, error) {
	v, err := crudview.New(crudview.Config{
		ParentID:  ID + ".assignments",
		Presenter: access.NewAssignmentsView(caller),
		IDs:       ids,
		NewRecord: func() model.Model {
			return &access.Assignment{}
		},
	})
	if err != nil {
		return nil, err
	}

	// Dynamically populate available role codes for the select dropdown
	out := &access.RoleRecordList{}
	caller.Call(access.ModelName+"."+access.OpListRoles, nil, out, func(err error) {
		if err != nil {
			return
		}
		opts := make([]fmt.KeyValue, 0, len(*out))
		for _, r := range *out {
			opts = append(opts, fmt.KeyValue{Key: r.Code, Value: r.Name})
		}
		v.SetOptions("role_code", opts...)
	})

	return platformd.NewUIModule(ID+".assignments", LabelAssignments, IconAssignments, v), nil
}

// AuditBrowser builds the audit trail UI module (read-only list).
func AuditBrowser(caller router.Caller, ids model.IDGenerator) (platformd.UIModule, error) {
	v, err := crudview.New(crudview.Config{
		ParentID:  ID + ".audit",
		Presenter: access.NewAuditView(caller),
		IDs:       ids,
	})
	if err != nil {
		return nil, err
	}
	return platformd.NewUIModule(ID+".audit", LabelAudit, IconAudit, v), nil
}

// Browser builds all three access administration screens in standard order:
// Roles, Assignments, Audit.
func Browser(caller router.Caller, ids model.IDGenerator) ([]platformd.UIModule, error) {
	rolesMod, err := RolesBrowser(caller, ids)
	if err != nil {
		return nil, err
	}
	assignMod, err := AssignmentsBrowser(caller, ids)
	if err != nil {
		return nil, err
	}
	auditMod, err := AuditBrowser(caller, ids)
	if err != nil {
		return nil, err
	}
	return []platformd.UIModule{rolesMod, assignMod, auditMod}, nil
}
```

Stage git commit:
`git add -A && git commit -m "feat(ui): add crudview browser constructors for roles, assignments and audit"`

---

## Stage 4 — `seed/seed.go`

Create `seed/seed.go`:

```go
package seed

import (
	"webtyp.com/access"
)

// Load provisions initial demonstration roles and an assignment.
func Load(m *access.Module) error {
	actorID := "seed-system"

	if err := m.SaveRole(actorID, access.RoleRecord{
		Code:        "admin",
		Name:        "Administrator",
		Description: "Full administrative access",
	}); err != nil {
		return err
	}

	if err := m.SaveRole(actorID, access.RoleRecord{
		Code:        "reception",
		Name:        "Receptionist",
		Description: "Front desk and appointments access",
	}); err != nil {
		return err
	}

	if err := m.SaveRole(actorID, access.RoleRecord{
		Code:        "viewer",
		Name:        "Auditor / Viewer",
		Description: "Read-only inspection access",
	}); err != nil {
		return err
	}

	if _, err := m.Assign(actorID, "admin@example.com", "admin"); err != nil {
		return err
	}

	return nil
}
```

Stage git commit:
`git add -A && git commit -m "feat(seed): add initial seed data for access module"`

---

## Stage 5 — `web/browser/main.go` (Runnable Demo)

Create `web/browser/main.go` with build tag `//go:build wasm`:

```go
//go:build wasm

package main

import (
	"webtyp.com/access"
	"webtyp.com/access/migrate"
	"webtyp.com/access/seed"
	"webtyp.com/access/ui"
	"webtyp.com/auth"
	"webtyp.com/auth/authority"
	"webtyp.com/ddl"
	. "webtyp.com/dom"
	. "webtyp.com/html"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/rbac"
	"webtyp.com/router/loopback"
	"webtyp.com/storage/mem"
	"webtyp.com/unixid"
)

type demoUser struct{}

func (demoUser) UserName() string    { return "Demo Admin" }
func (demoUser) UserAvatar() string  { return "" }
func (demoUser) UserRoles() []string { return []string{"admin"} }

func main() {
	ids, err := unixid.NewUnixID()
	if err != nil {
		panic(err)
	}

	db := orm.New(mem.New())
	if c, ok := db.RawConn().(ddl.Compiler); ok && c != nil {
		_ = authority.Migrate(db.RawConn(), c)
		_ = rbac.Migrate(db.RawConn(), c)
		_ = migrate.Migrate(db.RawConn(), c)
	}

	authMod, err := authority.New(db, auth.Config{IDs: ids})
	if err != nil {
		panic(err)
	}
	rbacSvc, err := rbac.New(db)
	if err != nil {
		panic(err)
	}

	accessMod, err := access.New(access.Deps{
		DB:        db,
		IDs:       ids,
		Users:     authMod,
		Roles:     rbacSvc,
		ProjectID: "demo-project",
		Assurance: model.AssuranceSingleFactor,
	})
	if err != nil {
		panic(err)
	}

	if err := seed.Load(accessMod); err != nil {
		panic(err)
	}

	caller := loopback.ActingAs("demo-project", "admin-1", accessMod)

	views, err := ui.Browser(caller, ids)
	if err != nil {
		panic(err)
	}

	homeView := Div().Text("Access Administration Demo")
	homeMod := platformd.NewUIModule("home", "Home", ui.IconRoles, homeView)

	allMods := append([]platformd.UIModule{homeMod}, views...)

	p := platformd.New(platformd.Config{
		DefaultModuleID: "home",
		Modules:         allMods,
		User:            demoUser{},
	})

	Render("app", p)
	select {}
}
```

Stage git commit:
`git add -A && git commit -m "feat(web): add runnable in-browser demo"`

---

## Stage 6 — `tests/` (View, Presenter and UI tests)

Add tests in `tests/ui_test.go`:

```go
package tests

import (
	"testing"
	"webtyp.com/access"
	"webtyp.com/access/seed"
	"webtyp.com/access/ui"
	"webtyp.com/model"
	"webtyp.com/router/loopback"
)

func TestItemizers(t *testing.T) {
	r := &access.RoleRecord{Id: "1", Code: "admin", Name: "Admin"}
	it := r.Item()
	if it.ID != "1" || it.Label != "Admin" || it.Description != "admin" {
		t.Errorf("unexpected RoleRecord.Item(): %+v", it)
	}

	a := &access.Assignment{Id: "admin/u1", Email: "admin@test.com", RoleCode: "admin", Name: "Boss"}
	itA := a.Item()
	if itA.ID != "admin/u1" || itA.Label != "admin@test.com" || itA.Description != "Boss (admin)" {
		t.Errorf("unexpected Assignment.Item(): %+v", itA)
	}

	e := &access.AuditEntry{Id: "aud1", ActorEmail: "admin@test.com", Action: "role.save", Target: "admin"}
	itE := e.Item()
	if itE.ID != "aud1" || itE.Label != "admin@test.com" || itE.Description != "role.save admin" {
		t.Errorf("unexpected AuditEntry.Item(): %+v", itE)
	}
}

func TestUIBrowserModules(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)
	if err := seed.Load(m); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}

	caller := loopback.ActingAs("p1", "admin-1", m)
	mods, err := ui.Browser(caller, &seqIDs{})
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}

	if len(mods) != 3 {
		t.Fatalf("expected 3 modules (Roles, Assignments, Audit), got %d", len(mods))
	}

	expectedIDs := []string{"access.roles", "access.assignments", "access.audit"}
	for i, exp := range expectedIDs {
		if mods[i].ID() != exp {
			t.Errorf("module %d ID: expected %q, got %q", i, exp, mods[i].ID())
		}
		if mods[i].View() == nil {
			t.Errorf("module %d view is nil", i)
		}
	}
}

func TestPresentersReload(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)
	if err := seed.Load(m); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}

	caller := loopback.ActingAs("p1", "admin-1", m)

	rolesPres := access.NewRolesView(caller)
	rolesPres.Reload(func(err error) {
		if err != nil {
			t.Fatalf("roles reload: %v", err)
		}
		items := rolesPres.Items()
		if len(items) != 3 {
			t.Errorf("expected 3 seeded roles, got %d", len(items))
		}
	})

	assignPres := access.NewAssignmentsView(caller)
	assignPres.Reload(func(err error) {
		if err != nil {
			t.Fatalf("assignments reload: %v", err)
		}
		items := assignPres.Items()
		if len(items) != 1 {
			t.Errorf("expected 1 seeded assignment, got %d", len(items))
		}
	})

	auditPres := access.NewAuditView(caller)
	auditPres.Reload(func(err error) {
		if err != nil {
			t.Fatalf("audit reload: %v", err)
		}
		items := auditPres.Items()
		if len(items) == 0 {
			t.Errorf("expected audit entries from seed, got 0")
		}
	})
}
```

Stage git commit:
`git add -A && git commit -m "test: add tests for itemizers, presenters and ui.Browser"`

---

## Stage 7 — Translations & Dependencies

Run `go mod tidy`.
Run `langc sync .` to update `lang.json`.
Check if any translation key is missing in `lang.json` and fill it.

Stage git commit:
`git add -A && git commit -m "chore: synchronize translations and update go.mod"`

---

## Stage 8 — Documentation

Update `README.md` to document UI screen mounting with `ui.Browser(caller, ids)` alongside the domain module.

Stage git commit:
`git add -A && git commit -m "docs: document UI browser module and demo"`

---

## Stages table

| Stage | Files touched | What changes |
|---|---|---|
| Stage 1 | `view.go` | Itemizer implementations and Presenter constructors |
| Stage 2 | `ui/module.go`, `ui/svg.go` | Identity constants, labels, and SVG icon declarations |
| Stage 3 | `ui/browser.go` | CrudView constructors for Roles, Assignments, Audit |
| Stage 4 | `seed/seed.go` | Seed data for roles and demo administrator |
| Stage 5 | `web/browser/main.go` | Runnable in-browser demo |
| Stage 6 | `tests/ui_test.go` | Itemizer, presenter, and UI browser tests |
| Stage 7 | `lang.json`, `go.mod`, `go.sum` | Sync dictionary and dependencies |
| Stage 8 | `README.md` | Document UI browser integration |

---

## Delivery check

Run `gotest` once. All tests (native + WASM) must pass with vet and race clean.
Once `gotest` passes, stage all files (`git add -A`), commit and open the PR.
