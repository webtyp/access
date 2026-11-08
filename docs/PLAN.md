---
PLAN: "refactor!: access on crud tables — roles, assignments and people through its own repositories over rbac and auth; entry screens as data; seed.Demo"
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> Never edit `model_orm.go` or `model_crud.go`: a build error there means the hand-written code is wrong.

# Plan: `access` on crud tables (D14, D15, D17, D20, D21, D22 of `webtyp/docs/ARCHITECTURE_MASTER_PLAN.md`)

This plan replaces the earlier `docs/PLAN.md` of this repo (it targeted the crud contract D20 replaced; the owner deletes it).

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go install webtyp.com/ormc/cmd/ormc@latest
go get webtyp.com/model@latest webtyp.com/crud@latest webtyp.com/orm@latest webtyp.com/view@latest webtyp.com/appkit@latest webtyp.com/auth@latest webtyp.com/rbac@latest
```
These must contain `Definition.Allow`/`model.JoinKey` (model), `crud.Table`/`crud.Store`/`crud.Transactional` (crud), `orm.GlobalStore`
(orm), `view.EntryScreens`/`view.Entry`/`view.Demo`/`model.TableSlice` lists (view), `appkit.Infra`/`appkit.Find` (appkit), and `auth`
published by its D20 plan (`/me` as an Infra route, `authority.Module` on crud). If `go doc webtyp.com/crud Store` fails, stop. Remove every `replace`; `go mod tidy`; run `ormc`.

## Why

`access` is the admin screen of who may do what. Only `access_audit` is a table of this repo; roles and assignments live in `webtyp/rbac`
(its own tables and user cache) and people are `webtyp/auth` users. Writing rbac's tables directly would skip its cache and rules, so
(D21, Repository) `access` writes **its own `crud.Store`** per resource over the `Roles` and `Users` interfaces it already declares
(`module.go`), and mounts them with the same `Allow` + `Mount<Entity>` as any table. These stores are not transactional: crud
accepts one record per request on them (`crud.ErrBatchNotAtomic`), which matches how the screens save today.

## Tables, stores and rules

| Model | Store | `Allow` | Hooks |
|---|---|---|---|
| `RoleRecord` (`access_role`) | `roleStore` (new, over `Roles`) | `model.Create \| model.Read \| model.Update \| model.Delete` | `Committed`: audit `AuditRoleSave` / `AuditRoleDelete` with the actor `c.UserID` (logic of `Record`) |
| `Assignment` (`access_assignment`) | `assignmentStore` (new, over `Roles` + `Users`) | `model.Create \| model.Read \| model.Delete` | `Create`: exactly one of `user_id` / `email` (field error otherwise; checks of `AssignUser` / `AssignByEmail`); `Committed`: audit |
| `Person` (`access_person`) | `personStore` (new, over `Users` + `Roles`) | `model.Read` | none |
| `AuditEntry` (`access_audit`) | `orm.GlobalStore(d.DB, d.IDs)` (no `tenant_id`: one audit per installation) | `model.Read` | none |

With `Deps.Assurance == model.AssuranceMultiFactor`, every `<Entity>Rules` sets `MFA: model.Create | model.Read | model.Update | model.Delete`.

## Stage 1 — `model.go`

- `Allow` as above on the four definitions. `RoleRecord.user_count` and `Person.roles` `ServerOnly: true`; `AuditEntry` fields all `ServerOnly`.
- `Assignment.Id` is the composite wire id `model.JoinKey(role_code, user_id)` (D22; replaces `AssignmentIDSeparator`, which is deleted — `grep -rn "AssignmentIDSeparator" .` → empty).
- Run `ormc`: `model_crud.go` has `MountRoleRecord`, `MountAssignment`, `MountPerson`, `MountAuditEntry`.

## Stage 2 — `stores.go` (new): the three repositories

Each implements `crud.Store` (`ReadOne`, `ReadAll`, `Create`, `UpdateFields`, `Delete`, `NewID`) and **not** `crud.Transactional`.
`ReadAll` supports only what the screens send: `crud.Query.Conditions` made of `storage.Like` on the Item label field (search) and
`storage.Eq` on fields of the model; any other condition → `ErrUnsupportedCondition = fmt.Err("access: this list filters only by search
or equality")`. It filters, orders by id and applies `After`/`Limit` in memory (lists are small: roles and people of one installation).

| Store | `ReadOne(rec, id)` | `ReadAll` | `Create(rec)` | `UpdateFields(rec, id, fields)` | `Delete(rec, id)` |
|---|---|---|---|---|---|
| `roleStore` | `Roles.GetRoleByCode(project, id)` (id = code) + `RoleUserCount`; `rbac.IsRoleNotFound` → error of kind `model.ErrorNotFound` | `Roles.ListRoles` (body of today's `ListRoles`) | body of `SaveRole` for a new code (`CreateRole` with `NewID()`, then `SetRoleSessionTTL`) | allowed fields `name`, `description`, `session_ttl` (others → `crud.ErrInvalidField`): body of `SaveRole` for an existing code | `Roles.DeleteRoleByCode` |
| `assignmentStore` | split id with `model.SplitKey(id, 2)`; present when the user holds the role (`UsersInRole`) | body of today's `ListAssignments` | `user_id` set → body of `AssignUser`; `email` set → body of `AssignByEmail` (creates the auth user when missing); sets `rec.Id = model.JoinKey(code, userID)` | not allowed → `crud.ErrInvalidField` (Allow has no Update; never reached) | `Roles.RevokeRoleByCode` |
| `personStore` | `Users.UserByID` + its roles | body of today's `ListPeople` | — (Allow is Read only; return `crud.ErrInvalidField`) | — | — |

`NewID()` returns `d.IDs.NewID()`. The audit no longer runs inside `SaveRole`/`AssignUser`/`Revoke`: it is the `Committed` hook (`m.auditCommitted`), with the
actor from `c.UserID` (`Users.UserByID(c.UserID).Email` for `actor_email`), written through the audit `orm.GlobalStore`.

## Stage 3 — `ops.go`

```go
var _ crud.Module = (*Module)(nil)

func (m *Module) MountOperations(reg crud.Registry) {
	mfa := model.Action(0)
	if m.assurance == model.AssuranceMultiFactor {
		mfa = model.Create | model.Read | model.Update | model.Delete
	}
	MountRoleRecord(reg, m.roleStore, RoleRecordRules{Committed: m.auditRole, MFA: mfa})
	MountAssignment(reg, m.assignmentStore, AssignmentRules{Create: m.checkAssignment, Committed: m.auditAssignment, MFA: mfa})
	MountPerson(reg, m.personStore, PersonRules{MFA: mfa})
	MountAuditEntry(reg, m.auditStore, AuditEntryRules{MFA: mfa})
}
```
Delete every `Op*` constant, `SaveRolesArgs`, `AssignRolesArgs`, `IDsArgs`, every `op*` handler and the `webtyp.com/router` / `rpc` imports
(`grep -rn "reg.Operation\|router.Context\|SaveRolesArgs\|IDsArgs" --include=*.go . | grep -v _test` → empty). Keep `EnsureRole`,
`EnsureRoleFor` (setup used by apps) and `Record` (used by them).

## Stage 4 — `access.Infra` (`module/server.go`) — D14

```go
// Infra mounts the session routes of the users module (logout, me). The app lists it in AppServer.Infra.
//
//	Server: appkit.AppServer{Infra: []appkit.Infra{module.Infra}}
var Infra = appkit.Infra{Name: "auth", Build: func(d appkit.Deps) (router.APIModule, error) {
	return appkit.Find[*authority.Module](d)
}}
```
`module/server.go:159`: `var browser view.ModuleBrowser`.

## Stage 5 — entry screens are data (D17) — `module/browser.go`

`browser` becomes `view.ModuleBrowser`; `Screens: func(d view.BrowserDeps) []view.Screen`; `Entry: func(d view.BrowserDeps) view.EntryScreens`.
Delete `LoginScreen` and the import of `webtyp.com/layout/login`; nothing reads `d.Brand`. The return is exactly:

```go
return view.EntryScreens{
	Login:       view.Entry{Subtitle: textEnterKey, Form: loginF.Render(), Message: loginMessage},
	Setup:       view.Entry{Subtitle: textFirstSetup, Form: setupForm.Render(), Message: setupMessage},
	IdleSeconds: defaultIdleMinutes * 60,
}
```
with `textEnterKey lang.Text = "Enter your key to continue"` and `textFirstSetup lang.Text = "First setup: create the administrator"`.
If `loginF` failed to build, `Login.Form` stays `nil`. `view.go`: every `view.NewCallerLister` takes `func() model.TableSlice { return &<X>List{} }`
and its `view.Ops` has no `Allow`.

## Stage 6 — the demo is `seed.Demo` (D15)

New `seed/demo.go` (`//go:build wasm`), from today's `web/browser/main.go`: the same `mem` database and three migration steps run on the
connection, the same `authority`, `rbac`, `access` modules and seed call; it returns
`view.Demo{Label: "Access", Timezone: "America/Santiago", ActingAs: <the admin id today's main signs in as>, Modules: []crud.Module{authMod, accessMod}, Migrations: steps, Screens: func(d view.BrowserDeps) []view.Screen { return ui.Screens(d.Caller) }}`.
Delete `web/browser/main.go` (`test ! -e web`). After `go mod tidy`: `grep -rn "webtyp.com/layout" .` → empty, `go.mod` included.

## Stage 7 — tests (`tests/`, through `crud.LoopbackActingAs("admin-1", mod)`; rows in `docs/TESTS.md`)

| Test | Setup → call → expected |
|---|---|
| `TestRole_CreateReadDelete` | `access_role_c` `{code:"reception", name:"Reception"}` → read `{id:"reception"}` → delete → read → `model.ErrorNotFound` |
| `TestRole_UpdateOnlyEditableFields` | update `["code"]` → `crud.ErrInvalidField`; update `["session_ttl"]` → stored in rbac |
| `TestRole_BatchRejected` | create two roles in one request → `crud.ErrBatchNotAtomic` |
| `TestAssignment_ByUserAndByEmail` | create with `user_id` → id `reception/u1`; create with an unknown email → auth user created and assigned |
| `TestAssignment_UserAndEmailBoth` | both set → field error |
| `TestAssignment_DeleteRevokes` | delete ids `["reception/u1"]` → `UsersInRole` no longer lists u1 |
| `TestAudit_ActorFromCaller` | create a role as `admin-1` → `access_audit_r` newest first, `[0].actor_email` is admin-1's email |
| `TestPerson_ReadOnly` | `access_person_c` → unknown operation |
| `TestPeople_ResourceIsPerson` | `rpc.Routes("/api", nil, mod)` → `access_person_r` has resource `access_person` |
| `TestDemo_Valid` (`//go:build wasm`) | `seed.Demo()` → no error, `Validate()` nil |

## Stage 8 — translations and documentation

`langc sync .` (zero untranslated, zero reuse warnings). `README.md` (4 mandatory sections, ≤150 lines): operations table
(`access_role_r/_c/_u/_d`, `access_assignment_r/_c/_d`, `access_person_r`, `access_audit_r`), `access.Infra` in the app example, the
note that the people list needs `access_person`. `docs/ARCHITECTURE.md`: why roles and people have their own repositories (D21) and
why they save one record at a time.

## Delivery check

`gotest` passes; `langc sync .` clean; every grep of this plan empty. A tree that does not compile or whose tests did not run is a failed delivery.

| Stage | Files |
|---|---|
| 1 | `model.go`, `model_orm.go`/`model_crud.go` (generated) |
| 2 | `stores.go`, `crud.go`, `queries.go`, `audit.go` |
| 3 | `ops.go` |
| 4 | `module/server.go` |
| 5 | `module/browser.go`, `view.go` |
| 6 | `seed/demo.go`, delete `web/browser/main.go`, `go.mod` |
| 7 | `tests/*.go`, `docs/TESTS.md` |
| 8 | `lang.json`, `README.md`, `docs/ARCHITECTURE.md` |
