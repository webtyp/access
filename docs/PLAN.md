---
PLAN: "refactor!: access on crud tables — roles, assignments and people through its own repositories over rbac and auth; entry screens as data; seed.Demo"
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> **Publishing:** you publish only by OPENING THE PULL REQUEST (a `git push` from your sandbox does not reach GitHub). Open it after Stage 3 with whatever compiles and keep committing to it until Stage 6.
> **Never ask a yes/no question about progress**; ask only when a signature in this plan is wrong, and name the symbol.
> **Before you submit:** `git status` must be clean and `git diff --stat main` must list the stage files (definition.go, rule_*.go, call_*.go, store.go, module.go, tests, docs). A pull request that carries only go.mod, go.sum or docs/PLAN.md is a failed delivery: if that is what your submit shows, `git add -A`, commit and submit again.
> Never edit `definition_gen.go`: a build error there means the hand-written code is wrong. Never write `go.mod`/`go.sum` lines by hand.

# Plan: `access` on crud tables (D14, D15, D17, D21, D22, D29–D31, D52 of `webtyp/docs/ARCHITECTURE_MASTER_PLAN.md`)

This plan was rewritten on 2026-11-10 with the vocabulary that is published today (`definitionc`, `definition.go`, `Definition`/`Rules`, rule files); an earlier text used `ormc`, `model_crud.go` and `Mount<Entity>`, which no longer exist.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go install webtyp.com/definitionc/cmd/definitionc@v0.1.43
export PATH="$PATH:$(go env GOPATH)/bin"
go get webtyp.com/model@v0.2.20 webtyp.com/crud@v0.0.14 webtyp.com/orm@v0.12.33 webtyp.com/storage@v0.2.2 webtyp.com/router@v0.4.22 webtyp.com/definitionc@v0.1.43 webtyp.com/view@v0.6.74 webtyp.com/appkit@v0.2.36 webtyp.com/auth@v0.0.112
go mod tidy
```
`definitionc` is in `$(go env GOPATH)/bin`, not in the PATH of a fresh shell: put the `export PATH=…` in the SAME command. **`definitionc` rewrites `go.mod` to older versions every run: after EVERY run repeat the `go get` line and `go mod tidy`.** No `replace` in `go.mod`. If a module cannot be fetched, stop and say so in the PR; never invent a hash. **`auth` is published (v0.0.112)**: it keeps the contract (plain `auth.User`, `auth.Identity`, `auth.Session`) and this plan imports only that; never `webtyp.com/auth/authority`.

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

## Rules learned from the failed sessions of this wave (read before starting)

1. **Do not write, summarize or iterate on a plan.** This index and the stage files are the finished spec. Start Stage 1 now.
2. **Read files by ranges** (`sed -n 1,80p file`, then `sed -n 81,160p file`); never one huge `cat`: its output is cut.
3. **A field a rule uses is the one in `definition_gen.go`:** `grep -n "type <Entity> struct" -A25 definition_gen.go`. `ServerOnly` only blocks the CLIENT: a rule may set any field of its receiver and of rows it creates with `tx.Create`. A field that does not exist is a question for the PR (name it), never an invention.
4. **No python or regex over the source:** edit file by file; `gotest` after each stage.

## Stages (do them in order; commit after each)

| Stage | File |
|---|---|
| 1 — `definition.go` | [PLAN_STAGE_1_definition_go.md](PLAN_STAGE_1_definition_go.md) |
| 2 — `stores.go` (new): the three repositories | [PLAN_STAGE_2_stores_go.md](PLAN_STAGE_2_stores_go.md) |
| 3 — `ops.go` goes away; the rules | [PLAN_STAGE_3_ops_go_goes_away_the_rules.md](PLAN_STAGE_3_ops_go_goes_away_the_rules.md) |
| 4 — `access.Infra` (`module/server.go`) — D14 | [PLAN_STAGE_4_access_infra.md](PLAN_STAGE_4_access_infra.md) |
| 5 — entry screens are data (D17) — `module/browser.go` | [PLAN_STAGE_5_entry_screens_are_data.md](PLAN_STAGE_5_entry_screens_are_data.md) |
| 6 — the demo is `seed.Demo` (D15) | [PLAN_STAGE_6_the_demo_is_seed_demo.md](PLAN_STAGE_6_the_demo_is_seed_demo.md) |

## Delivery check (part 1 of 2)

This plan is only Stages 1–6; the files `PLAN_STAGE_7_*.md` and later belong to the NEXT plan: ignore them. In this part you ALSO delete every test in `tests/` that calls a method or constant this migration removes (`go vet ./tests/` lists them) together with its row in `docs/TESTS.md`; the new tests are written in part 2. `gotest` passes on every destination (it builds, vets, races and runs every destination) — the root module must compile with no test of the removed code left. A tree that does not compile is a failed delivery. Submit as soon as it is green (git add -A; `git diff --stat main` lists the stage files).
