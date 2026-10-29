---
PLAN: "refactor(ui): Screens(caller) []view.Screen — access declares its screens as data like every module; no platformd/crudview in ui"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 12404899667519076581
PR: https://github.com/webtyp/access/pull/5
REVIEWED_COMMIT: 46fe0d560a82739ed7164381d84a8005044fb57a
---

> Read AGENTS.md at the repo root before any change.

# PLAN — `access` screens as `view.Screen`

## Why

Every module now declares its screens as data (`ui.Screens(caller) []view.Screen`) and the app
draws them with `layout/app.BuildMenu`, which also fills each select from the screen that lists its
`Options` model (`layout/app/menu.go:52-80`). `access/ui/browser.go` still builds
`platformd.UIModule`s by hand and fills the role select with its own `caller.Call`. So
`veltylabs/mjosefa-cms` cannot list the users-and-roles screens it needs to work from anywhere
(decision D20 of `veltylabs/mjosefa-cms/docs/MVP_DEPLOYMENT_ROADMAP.md`, phase M2j). Same pattern
as `device_manager/ui/browser.go` (reference). Also: two menu labels are Spanish keys
(`"Asignaciones"`, `"Auditoría"`), against the **translations** rule (English keys, translated in
`lang.json`).

## APIs used

| Symbol | Exact | Tag |
|---|---|---|
| `view.Screen` | `struct { ID string; Label lang.Text; Icon string; Permission model.Resource; Presenter Presenter; Component dom.Component; Help lang.Text; Children []Screen }` | view (already in go.mod) |
| `view.WithDescription` | `func WithDescription(d lang.Text) Option` | view |
| `view.New` | `func New(b …, rec …, opts ...Option) Presenter` (as called in `view.go:27`) | view |
| `app.BuildMenu` | `func BuildMenu(screens []view.Screen, ids model.IDGenerator, allows func(model.Resource) bool) (Menu, error)`; `Menu{Modules []layout.UIModule; CanView func(id string) bool}` | layout (demo only) |

## Stage 1 — `view.go`: descriptions

`NewRolesView`, `NewAssignmentsView`, `NewAuditView` get a trailing `opts ...view.Option` passed to
`view.New` after `view.WithTitle(...)`. Callers that pass none are unchanged.

## Stage 2 — `model.go`: the role select fills itself

In `AssignmentModel`, field `role_code` gets `Options: &RoleRecordModel` (a role's `Id` equals its
`Code`, `model.go:28`, so the select stores the code). Regenerate with `ormc` if the field list is
mirrored in `model_orm.go` (run the generator the repo already uses; do not hand-edit generated code).

## Stage 3 — `ui/`

- `ui/module.go`: `LabelAssignments = "Assignments"`, `LabelAudit = "Audit"` (English keys). Add
  help texts as `lang.Text` constants (English, one sentence each, saying what the screen is for):
  `descRoles`, `descAssignments`, `descAudit`.
- `ui/browser.go`: replace everything with

  ```go
  // Screens declares the screens access offers to the app's menu.
  //
  //	screens := ui.Screens(caller)
  func Screens(caller router.Caller) []view.Screen {
  	return []view.Screen{
  		{ID: ID + ".roles", Label: LabelRoles, Icon: string(IconRoles), Permission: access.ResourceRole,
  			Presenter: access.NewRolesView(caller, view.WithDescription(descRoles))},
  		{ID: ID + ".assignments", Label: LabelAssignments, Icon: string(IconAssignments), Permission: access.ResourceAssignment,
  			Presenter: access.NewAssignmentsView(caller, view.WithDescription(descAssignments))},
  		{ID: ID + ".audit", Label: LabelAudit, Icon: string(IconAudit), Permission: access.ResourceAudit,
  			Presenter: access.NewAuditView(caller, view.WithDescription(descAudit))},
  	}
  }
  ```

  Delete `RolesBrowser`, `AssignmentsBrowser`, `AuditBrowser`, `Browser` and the manual
  `SetOptions` call. `ui` no longer imports `layout/crudview` nor `layout/platformd`.
- `lang.json`: add the new keys (`Assignments`, `Audit`, the three descriptions) translated to every
  language the file already has; run `langc sync .` → 0 pending, 0 reuse warnings.

## Stage 4 — `web/browser/main.go` (the demo)

Build the menu as the apps do: `menu, err := app.BuildMenu(ui.Screens(caller), ids, func(model.Resource) bool { return true })`
(panic on error), then `platformd.Platform{DefaultID: ID + ".roles", Modules: menu.Modules, CanView: menu.CanView, User: demoUser{}}`
(keep the existing `demoUser`; drop the hand-made `home` module). Open the demo with `webtyp` at
the repo root, take a screenshot of the three screens, and attach it to the PR description.

## Stage 5 — tests

- Any test calling `ui.Browser`/`RolesBrowser`/… calls `ui.Screens(caller)` instead.
- New `tests/screens_test.go` (stdlib, no tag): `ui.Screens(caller)` returns 3 screens; each passes
  `Validate()` (the method `view.Screen` already has); permissions are `access.ResourceRole`,
  `ResourceAssignment`, `ResourceAudit`; `AssignmentModel`'s `role_code` has `Options` pointing to
  `RoleRecordModel`.

## Stage 6 — docs

`README.md` and `docs/ARCHITECTURE.md`: the app lists `ui.Screens(caller)` in its registry; the role
select is filled by `BuildMenu` from the roles screen.

## Delivery check

```
gotest
```

- A PR that does not compile, or whose tests could not run, is a failed delivery.
- Never pause to ask permission: when the check passes, commit, push and open the PR. Stuck 30
  minutes: open the PR listing what fails.
- `git diff --stat main` lists only: `view.go`, `model.go` (and its generated file if regenerated),
  `ui/*`, `web/browser/main.go`, `lang.json`, `tests/*`, `README.md`, `docs/ARCHITECTURE.md`,
  `go.mod`, `go.sum`, this plan.

## Executor notes

- Taking a screenshot of the browser demo with `webtyp` wasn't able to be completed, as the `webtyp` command line tool wasn't available in the test environment.
- Generating code with `ormc` and `langc` encountered some friction where they needed to be built and installed in order for the execution environment to locate them properly.
- All functional changes to `model.go`, `view.go`, `ui/browser.go`, `ui/module.go`, `web/browser/main.go`, `tests/` and translations via `lang.json` were completed.

## Stages

| # | Files | Done when |
|---|---|---|
| 1 | `view.go` | compiles |
| 2 | `model.go` | compiles |
| 3 | `ui/*`, `lang.json` | compiles; `langc sync .` clean |
| 4 | `web/browser/main.go` | demo shows the 3 screens (screenshot in PR) |
| 5 | `tests/*` | suite green |
| 6 | docs | sections present |
