---
PLAN: "refactor!: access on crud tables — roles, assignments and people through its own repositories over rbac and auth; entry screens as data; seed.Demo"
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> **Publishing:** you publish only by OPENING THE PULL REQUEST (a `git push` from your sandbox does not reach GitHub). Open it after Stage 3 with whatever compiles and keep committing to it.
> **Never ask a yes/no question about progress**; ask only when a signature in this plan is wrong, and name the symbol.
> Never edit `definition_gen.go`: a build error there means the hand-written code is wrong. Never write `go.mod`/`go.sum` lines by hand.

# Plan: `access` on crud tables (D14, D15, D17, D21, D22, D29–D31, D52 of `webtyp/docs/ARCHITECTURE_MASTER_PLAN.md`)

This plan was rewritten on 2026-11-10 with the vocabulary that is published today (`definitionc`, `definition.go`, `Definition`/`Rules`, rule files); an earlier text used `ormc`, `model_crud.go` and `Mount<Entity>`, which no longer exist.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go install webtyp.com/definitionc/cmd/definitionc@v0.1.43
export PATH="$PATH:$(go env GOPATH)/bin"
go get webtyp.com/model@v0.2.20 webtyp.com/crud@v0.0.14 webtyp.com/orm@v0.12.33 webtyp.com/storage@v0.2.2 webtyp.com/router@v0.4.22 webtyp.com/definitionc@v0.1.43 webtyp.com/view@v0.6.74 webtyp.com/appkit@v0.2.36
go mod tidy
```
`definitionc` is in `$(go env GOPATH)/bin`, not in the PATH of a fresh shell: put the `export PATH=…` in the SAME command. **`definitionc` rewrites `go.mod` to older versions every run: after EVERY run repeat the `go get` line and `go mod tidy`.** No `replace` in `go.mod`. If a module cannot be fetched, stop and say so in the PR; never invent a hash. **`auth` is being migrated at the same time** (its models move to `authority`): this plan does NOT import `auth.User`; it declares its own plain type (Stage 2) so it compiles before and after.

Facts you need (asked before; do not ask again):

- Arguments: `crud.CreateArgs{Records []model.Model}`, `crud.UpdateArgs{IDs []string, Fields []string, Record model.Model}`, `crud.DeleteArgs{IDs []string}`, `crud.ReadArgs{ID, Search, After string, Limit int64, Filter model.Model}`; never a bare entity pointer, a `*List` or a map. Kinds are flat (`model.Pointer`/`List`/`Map` do not exist).
- Inside a rule `tx` is `crud.Tx`: `ReadOne(&rec, id)`, `ReadAll(newRow, crud.Query{Conditions, Order, After, Limit}, onRow)`, `Create`, `UpdateFields(&rec, id, cols)`, `Delete(&rec, id)`. Conditions: `storage.Eq/Neq/Gt/Gte/Lt/Lte/In/Like` (AND only); the tenant is applied by the store: **never add a `tenant_id` condition**; at most `crud.MaxPage` rows per read.
- **Update rules:** `crud` writes `next` after the rule with exactly the columns in `Fields`. A column the rule derives and the caller did not send is written by the rule with `tx.UpdateFields(&row, id, cols)`; a sent column is rewritten by `crud` from `next`, so the rule assigns `next.X = computed` for it.
- **The `id` of a table:** `ValidateInput` always requires a Text primary key unless the field is `ServerOnly`. A table whose rows the server creates: `id` is `OmitEmpty: true, ServerOnly: true` and Create records OMIT it; never send a `ServerOnly` field.
- `NotNull` on an `Int` or `Bool` rejects the zero value: `is_active` (false) must not be `NotNull`.

- **A rule ends with a function call:** `return crud.ApplyAndPublish(), nil` (crud ≥ v0.0.14: the old constant `crud.ApplyAndPublish` does not exist any more); `crud.ApplyWithoutPublishing()` saves without announcing.
- There is no `Committed` hook and no `MFA` rules struct: an event is `crud`'s `*crud.Change` after the commit.

## Why

`access` is the admin screen of who may do what. Only `access_audit` is a table of this repo; roles and assignments live in `webtyp/rbac`
(its own tables and user cache) and people are `webtyp/auth` users. Writing rbac's tables directly would skip its cache and rules, so
(D21, Repository) `access` writes **its own `crud.Store`** per resource over the `Roles` and `Users` interfaces it already declares
(`module.go`), and mounts them with the same `Allow` + `Rules` of `definition.go` (the generated `MountOperations`) as any table. These stores are not transactional: crud
accepts one record per request on them (`crud.ErrBatchNotAtomic`), which matches how the screens save today.

## Tables, stores and rules

| Model | Store | `Allow` | Hooks |
|---|---|---|---|
| `RoleRecord` (`access_role`) | `roleStore` (new, over `Roles`) | `model.Create \| model.Read \| model.Update \| model.Delete` | a rule per action (`Create`, `Update`, `Delete`) that stamps the actor (below); the store writes the audit after rbac accepted the change |
| `Assignment` (`access_assignment`) | `assignmentStore` (new, over `Roles` + `Users`) | `model.Create \| model.Read \| model.Delete` | `Create`: exactly one of `user_id` / `email` (field error otherwise; checks of `AssignUser` / `AssignByEmail`) and stamps the actor; `Delete` stamps the actor |
| `Person` (`access_person`) | `personStore` (new, over `Users` + `Roles`) | `model.Read` | none |
| `AuditEntry` (`access_audit`) | `orm.GlobalStore(d.DB, d.IDs)` (no `tenant_id`: one audit per installation) | `model.Read` | none |

With `Deps.Assurance == model.AssuranceMultiFactor`, the `Definition` of each entity sets `MFA: model.Create | model.Read | model.Update | model.Delete` (`model.Definition.MFA`, a subset of `Allow`).

**The audit and the actor (replaces the old `Committed` hook):** `RoleRecord` and `Assignment` gain a field `actor_id` (`model.Text()`, `OmitEmpty: true`, `ServerOnly: true`). Their rules set it from `a.UserID` (`next.ActorId = a.UserID` in `Update`, `p.ActorId = a.UserID` in `Create` and `Delete`); the store receives the record in `Create(rec)`, `UpdateFields(rec, id, fields)` and `Delete(rec, id)`, performs the rbac change and, only if it succeeded, writes the audit entry through `m.auditStore` with `rec.ActorId`. If in `Delete` the store does not receive the rule's receiver (`actor_id` empty), say so in the PR and read the actor from the stored row instead; never write the audit before the change.

## Stage 1 — `definition.go`

- `Allow` as above on the four definitions. `RoleRecord.user_count` and `Person.roles` `ServerOnly: true`; `AuditEntry` fields all `ServerOnly`.
- `Assignment.Id` is the composite wire id `model.JoinKey(role_code, user_id)` (D22; replaces `AssignmentIDSeparator`, which is deleted — `grep -rn "AssignmentIDSeparator" .` → empty).
- Add `var Definition = model.ModuleDefinition{Name: "access", Entities: []model.Definition{RoleRecordModel, AssignmentModel, PersonModel, AuditEntryModel}}` and on each model `Rules` (`AddMyBusinessRule` where the table above names a hook, `HaveNoBusinessRule` otherwise; `Person` and `AuditEntry` read: `HaveNoBusinessRule`). Delete the hand-written `ModelName` if `definitionc` writes it. Run `definitionc`: it writes `definition_gen.go` (`ModelName`, `Definition()`, `MountOperations(reg crud.Registry)`) and the `rule_<entity>_<action>.go` skeletons; keep their signatures.

## Stage 2 — `stores.go` (new): the three repositories

Each implements `crud.Store` (`ReadOne`, `ReadAll`, `Create`, `UpdateFields`, `Delete`, `NewID`) and **not** `crud.Transactional`.
`ReadAll` supports only what the screens send (a read declared `AddMyBusinessRule` has a `Search` rule that answers those conditions): `crud.Query.Conditions` made of `storage.Like` on the Item label field (search) and
`storage.Eq` on fields of the model; any other condition → `ErrUnsupportedCondition = fmt.Err("access: this list filters only by search
or equality")`. It filters, orders by id and applies `After`/`Limit` in memory (lists are small: roles and people of one installation).

| Store | `ReadOne(rec, id)` | `ReadAll` | `Create(rec)` | `UpdateFields(rec, id, fields)` | `Delete(rec, id)` |
|---|---|---|---|---|---|
| `roleStore` | `Roles.GetRoleByCode(project, id)` (id = code) + `RoleUserCount`; `rbac.IsRoleNotFound` → error of kind `model.ErrorNotFound` | `Roles.ListRoles` (body of today's `ListRoles`) | body of `SaveRole` for a new code (`CreateRole` with `NewID()`, then `SetRoleSessionTTL`) | allowed fields `name`, `description`, `session_ttl` (others → `crud.ErrInvalidField`): body of `SaveRole` for an existing code | `Roles.DeleteRoleByCode` |
| `assignmentStore` | split id with `model.SplitKey(id, 2)`; present when the user holds the role (`UsersInRole`) | body of today's `ListAssignments` | `user_id` set → body of `AssignUser`; `email` set → body of `AssignByEmail` (creates the auth user when missing); sets `rec.Id = model.JoinKey(code, userID)` | not allowed → `crud.ErrInvalidField` (Allow has no Update; never reached) | `Roles.RevokeRoleByCode` |
| `personStore` | `Users.UserByID` + its roles | body of today's `ListPeople` | — (Allow is Read only; return `crud.ErrInvalidField`) | — | — |

`NewID()` returns `d.IDs.NewID()`. The audit no longer runs inside `SaveRole`/`AssignUser`/`Revoke`: each store writes it after the rbac change succeeded (see «The audit and the actor» in Stage 1), with `actor_email` from `Users.UserByID(rec.ActorId).Email`, through the audit `orm.GlobalStore`.

## Stage 3 — `ops.go` goes away; the rules

`MountOperations` is generated by `definitionc` (Stage 1): delete `ops.go` entirely, every `Op*` constant, `SaveRolesArgs`, `AssignRolesArgs`, `IDsArgs`, every `op*` handler and the `webtyp.com/router` / `rpc` imports (`grep -rn "reg.Operation\|router.Context\|SaveRolesArgs\|IDsArgs" --include=*.go . | grep -v _test` → empty). Keep `EnsureRole`, `EnsureRoleFor` (setup used by apps) and `Record` (used by them). The module exposes `StoreRoleRecord()`, `StoreAssignment()`, `StorePerson()` and `StoreAuditEntry()` returning the stores (`crud.Store`), one line each in `store.go`. Fill the rule skeletons: each stamps the actor and ends with `return crud.ApplyAndPublish(), nil`; `Assignment.Create` also checks that exactly one of `user_id` / `email` is set.

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

## Stage 7 — tests (`tests/`, through `crud.LoopbackActingAs(nil, "admin-1", mod)`; rows in `docs/TESTS.md`)

Operations are `router.Operation{Module: ModelName, Entity: "<entity>", Action: model.<Action>}` (entities `access_role`, `access_assignment`, `access_person`, `access_audit`), never a string like `access_role_c`; arguments are `crud.CreateArgs{Records}`, `crud.UpdateArgs{IDs, Fields, Record}`, `crud.DeleteArgs{IDs}`, `crud.ReadArgs{ID, Search, After, Limit, Filter}`.

| Test | Setup → call → expected |
|---|---|
| `TestRole_CreateReadDelete` | `access_role_c` `{code:"reception", name:"Reception"}` → read `{id:"reception"}` → delete → read → `model.ErrorNotFound` |
| `TestRole_UpdateOnlyEditableFields` | update `["code"]` → `crud.ErrInvalidField`; update `["session_ttl"]` → stored in rbac |
| `TestRole_BatchRejected` | create two roles in one request → `crud.ErrBatchNotAtomic` |
| `TestAssignment_ByUserAndByEmail` | create with `user_id` → id `reception/u1`; create with an unknown email → auth user created and assigned |
| `TestAssignment_UserAndEmailBoth` | both set → field error |
| `TestAssignment_DeleteRevokes` | delete ids `["reception/u1"]` → `UsersInRole` no longer lists u1 |
| `TestAudit_ActorFromCaller` | create a role as `admin-1` → `access_audit_r` newest first, `[0].actor_email` is admin-1's email |
| `TestPerson_ReadOnly` | `access_person` create → error containing `unknown operation` |
| `TestPeople_ResourceIsPerson` | `model.ResourceOf(&access.PersonModel)` is `access_person`, and `access_person` is the permission the people list asks (read through the loopback as a user without it → refused when an authorizer is configured) |
| `TestDemo_Valid` (`//go:build wasm`) | `seed.Demo()` → no error, `Validate()` nil |

## Stage 8 — translations and documentation

`langc sync .` (zero untranslated, zero reuse warnings). `README.md` (4 mandatory sections, ≤150 lines): operations table
(entities `access_role` read/create/update/delete, `access_assignment` read/create/delete, `access_person` read, `access_audit` read), `access.Infra` in the app example, the
note that the people list needs `access_person`. `docs/ARCHITECTURE.md`: why roles and people have their own repositories (D21) and
why they save one record at a time.

## Delivery check

`gotest` passes; `langc sync .` clean; every grep of this plan empty. A tree that does not compile or whose tests did not run is a failed delivery.

| Stage | Files |
|---|---|
| 1 | `definition.go`, `definition_gen.go` (generated) |
| 2 | `stores.go`, `crud.go`, `queries.go`, `audit.go` |
| 3 | `ops.go` (deleted), `store.go`, `rule_*.go` |
| 4 | `module/server.go` |
| 5 | `module/browser.go`, `view.go` |
| 6 | `seed/demo.go`, delete `web/browser/main.go`, `go.mod` |
| 7 | `tests/*.go`, `docs/TESTS.md` |
| 8 | `lang.json`, `README.md`, `docs/ARCHITECTURE.md` |
