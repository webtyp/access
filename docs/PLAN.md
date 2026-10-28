---
PLAN: "feat!: list rows declared on the model fields (Row/Options); Item() removed"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 6968692468430378003
PR: https://github.com/webtyp/access/pull/3
REVIEWED_COMMIT: 07e040c33ed0cb98502364bd62dfb6e17147e52b
---

> Read AGENTS.md at the repo root before any change.

<!-- Dispatch only after webtyp.com/model v0.2.7, webtyp.com/view v0.6.47 are published. -->

# Plan — `access`: what is saved shows everywhere

## Why

`webtyp.com/view` v0.6.47 keeps one shared copy of each record per page and builds every list row from the
record's fields (`model.RowOf`, `webtyp.com/model` v0.2.7), so what a person saves shows at once in every list and
select that displays it. The hand-written `Item()` methods and `view.Itemizer` are gone: each listed model marks on
its fields where they show in the row (`Row: model.RowTitle` / `RowDetail` / `RowBadge`), and a select says which
list fills it with `Ref` (foreign key) or `Options` (no foreign key). `view.Lookup` is gone too. Breaking change,
no compatibility layer.

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go install webtyp.com/ormc/cmd/ormc@latest
go get webtyp.com/model@v0.2.7
go get webtyp.com/view@v0.6.47
```

If a tag lacks a symbol of «APIs used», stop and say so in the PR.

## Ecosystem search

Nothing new is created: `model.RowSlot`, `Field.Row`, `Field.Options`, `model.RowOf` (model v0.2.7) and
`view.Records` (view v0.6.47) are reused.

## Design gate

1. **Prior art.** Laravel Nova (`->showOnIndex()` on the field), Strapi «main field», Payload `useAsTitle`:
   the row is declared on the fields.
2. **Novice-name test.** `Row: model.RowTitle` — «in the row, this is the title».
3. **Complexity ledger.** −3 `Item()` method(s); +1 `Row:` per shown field.
4. **Where it belongs.** The row is a fact about the model (`model.go`).
5. **What it deletes.** Every `Item()` below.

## APIs used

| Symbol | Exact signature | Tag | File |
|---|---|---|---|
| `model.Field.Row` | `Row RowSlot`; constants `model.RowTitle`, `model.RowDetail`, `model.RowBadge` | model v0.2.7 | `field.go`, `row.go` |
| `model.Field.Options` | `Options *Definition` (a select fed by another list without a foreign key) | model v0.2.7 | `field.go` |
| `model.RowOf` | `func RowOf(f Fielder) Row`; `Row{ID, Title, Detail, Badge string}`; several `RowDetail` join with `" · "` in declaration order | model v0.2.7 | `row.go` |
| `view.NewRecords` / `(*Records).Caller` | `func NewRecords() *Records`; `func (r *Records) Caller(c router.Caller) router.Caller` — `view.NewCallerLister` panics with any other caller | view v0.6.47 | `records.go` |
| `view.Screen` | leaf: `Presenter` **or** `Component dom.Component` (+ `Help`); no `Lookups` | view v0.6.47 | `screen.go` |

If a signature in the published tag differs from this table, stop and name the symbol in the PR; never adapt it by guessing.


## Stage 1 — mark the row on the fields (`model.go`), then regenerate

These are the `Item()` methods of this repo today (`view.go`), verbatim:

```go
func (r *RoleRecord) Item() view.Item {
	return view.Item{
		ID:          r.Id,
		Label:       r.Name,
		Description: r.Code,
	}
}

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
```

For each one, in the `model.Definition` of that type (`<Type>Model` in `model.go`):

- the field read into `Label` → `Row: model.RowTitle`;
- each field read into `Description` → `Row: model.RowDetail` (several, joined with « · » in field order: if the
  method concatenates them in another order, reorder nothing — say so in the PR);
- the field read into `LeadTop`/`LeadMain`/`LeadBottom` (a date, an hour) → `Row: model.RowBadge` on that ONE
  field (the widget formats it now);
- `ID` must be the primary-key field (`DB: &model.FieldDB{PK: true}`); if it is not, stop and say so in the PR.
- A text that is not a plain field (a translated status, a computed label, a constant, a call) cannot be declared:
  mark the field it is computed from and list each such case in the PR.

Then run `ormc` (it rejects an invalid row: two `RowTitle`, details without a title, `Ref` with `Options`). Never
edit `model_orm.go` by hand.

## Stage 2 — delete the `Item()` methods (`view.go`)

Delete every function shown in Stage 1 and any `var _ view.Itemizer = …` line, plus helpers only they used (e.g.
date-abbreviation arrays). Check: `grep -rn "Item() view.Item\|view.Itemizer\|LeadTop\|LeadMain\|LeadBottom" --include=*.go .` → empty.

## Stage 3 — selects
No `Lookups` in this repo: nothing to do.
Check: `grep -rn "view.Lookup\|Lookups:" --include=*.go .` → empty.

## Stage 4 — saving answers the saved record

Every operation named as `Save` in a `view.Ops{…}` of this repo must answer the record it stored
(`ctx.Encode(&out)` with the saved record, the last one for a batch) — `view` now puts that answer in the shared
copy. Read each handler in `ops.go`; add the `ctx.Encode` where it is missing and name those handlers in the PR.

## Stage 5 — tests

- Every test that called `Item()` asserts `model.RowOf(&rec)` instead (`Title`/`Detail`/`Badge`).
- Every test that builds a presenter over `view.NewCallerLister(caller, …)` wraps the caller once:
  `view.NewRecords().Caller(caller)`.
- Add `TestRowsDeclaredOnFields` in `tests/`: for each listed model, build one record with known values and assert
  the exact `model.RowOf` result (one table row per model).

## Stage 6 — docs and translations

`README.md`/`docs/*`: the row is declared on the fields; no `Item()`. `langc sync .` and fill every empty value;
zero untranslated keys, zero reuse warnings.

## Delivery

- `gotest` green. A PR that does not compile, or whose tests could not run, is a failed delivery.
  Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`, `go mod tidy` before `ormc`.
- Never edit a generated file (`model_orm.go`): a build error there means the hand-written code is wrong.
- Never pause to ask permission or report progress: when the check passes, commit, push and open the PR. Stuck 30
  minutes → open the PR listing what fails. Commit after every stage; never `git reset`, `git restore`,
  `git checkout -- <file>` or `git stash` on your own work.

| Stage | File | Action |
|---|---|---|
| 1 | `model.go`, `model_orm.go` (regenerated), `go.mod`, `go.sum` | `Row:`/`Options:` on the fields |
| 2 | `view.go` | `Item()` gone |
| 3 | `ui/browser.go` | `Lookups` gone |
| 4 | `ops.go` | save answers the record |
| 5 | `tests/*` | `model.RowOf` |
| 6 | docs, comments, `lang.json` | translate |

## Executor notes

- `Assignment` concatenation order changed (`RoleCode` then `Name` vs `Name` then `RoleCode` with parens).
- `AuditEntry` concatenation order and formatting changed (added middot instead of dash).
- Test failures occurring on dependency `webtyp.com/layout` version `v0.3.77` since it relies on the now-deleted `view.Itemizer`. Upgraded `view` to `v0.6.47` as requested which removed `Itemizer`, resulting in this incompatibility in the current available version of `layout`. Wait for `webtyp.com/layout@v0.3.78` as per instructions or ignore this error.
