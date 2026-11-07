---
PLAN: "feat!: paged lists searched on the server (view v0.6.64)"
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work.

# access — paged lists searched on the server

`webtyp.com/view` v0.6.64 changed its list contract. Until this repo follows it, its list screens answer 400
("cannot read the request") because the view now sends `view.ListArgs` to the list operation. The full recipe is
restated below; source: `https://github.com/webtyp/view/blob/main/docs/LIST_PAGING.md`.

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go get webtyp.com/view@v0.6.64
go get webtyp.com/layout@latest   # the release that moved crudview to view v0.6.64
```

Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`, `go mod tidy` after `go get`.

## Sites in this repo (found by grep — change every one)

`view.Ops` declarations (each list operation named in `List:` accepts `view.ListArgs`; each `Ops` gets its `Page`):

```
./view.go:20:		view.Ops{Module: ModelName, List: OpListPeople},
./view.go:29:		view.Ops{
./view.go:43:		view.Ops{
./view.go:57:		view.Ops{
```

Hand-written `view.Lister` implementations (new `List(args view.ListArgs, done …)` and `Page()`):

```
(none)
```

Calls to the deleted `Presenter.Filter` (→ `Search(term, nil)` then `Items()`):

```
(none)
```

## Page of each list

| List operation | `Page` in its `view.Ops` |
|---|---|
| `OpListAudit` | `view.Page{First: 50, Size: 50, More: true}` |
| every other list | `view.Page{First: view.All}` |

## Recipe

`webtyp.com/view` v0.6.63 replaced "list everything, filter in the browser" with **paged lists declared by the
module and searched on the server**. A module that shows a list with `view.NewCallerLister` or implements
`view.Lister` makes the changes below. Contract: [page.go](../page.go), [lister.go](../lister.go),
[view.go](../view.go).

## What changed in view

| Before | Now |
|---|---|
| `Lister.List(done func(rows []view.Record, err error))` | `Lister.List(args view.ListArgs, done func(rows []view.Record, err error))` and `Lister.Page() view.Page` |
| The list operation got no body | It gets `view.ListArgs{Search, After, Limit}` and **must accept it** (otherwise it answers 400 and the screen lists nothing) |
| `Presenter.Filter(term) []model.Item` (local) | `Presenter.Search(term, done)`, `Presenter.HasMore()`, `Presenter.More(done)` (server) |
| — | `view.Ops.Page` (`view.Page{First, Size, More}`), `view.All`, `view.DefaultPage`, `view.MaxPage`, `view.Cap` |

## 1. The list operation accepts `view.ListArgs`

Every operation named in `view.Ops.List` declares and decodes `view.ListArgs`, caps the limit, searches on the
server and continues after the last row shown. Rows come in **ascending id order** (ids are unixid strings of equal
length, so string order is time order), the same order every call, so `After` continues it.

```go
reg.Operation(OpListThings, m.opListThings).Requires(model.ResourceOf(&ThingModel), model.Read).Accepts(&view.ListArgs{})

func (m *Module) opListThings(ctx router.Context) {
	var args view.ListArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead) // the module's existing "cannot read" error
		return
	}
	rows, err := m.listThings(args.Search, args.After, view.Cap(args.Limit))
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	var list ThingList
	for i := range rows {
		list = append(list, &rows[i])
	}
	_ = ctx.Encode(&list)
}

// listThings: tenant from the module (m.tenantID), never from the request.
func (m *Module) listThings(search, after string, limit int64) ([]Thing, error) {
	var t Thing
	qb := m.db.Query(&t).Where(Thing_.TenantId).Eq(m.tenantID)
	if search != "" {
		qb = qb.Where(Thing_.Name).Like("%" + search + "%") // the field marked Item: model.ItemLabel
	}
	if after != "" {
		qb = qb.Where(Thing_.Id).Gt(after)
	}
	qb = qb.OrderBy(Thing_.Id).Asc().Limit(int(limit))
	rows, err := ReadAllThing(qb)
	…
}
```

- **Never use `qb.Or()`**: `orm` does not group OR conditions, so `tenant = X AND a LIKE s OR b LIKE s` would leak
  rows of other tenants. Search one field (the `model.ItemLabel` one); a module that needs several fields picks the
  field by the shape of the term (e.g. a valid RUT → the RUT column, compared with `input.NormalizeRUT`).
- A model without a tenant column drops that `Where`. A model without `Id` orders by its primary key.
- If the operation already had arguments of its own (filters), those stay in the module's other operations; the
  view's list operation receives only `view.ListArgs`.

## 2. The module declares its Page

In `view.Ops`:

```go
view.Ops{Module: ModelName, List: OpListThings, Save: OpUpsertThing, Page: view.Page{First: view.All}}
```

- Small or bounded lists (catalogs, configuration, staff, devices, zones): `view.Page{First: view.All}` (every
  row up to `view.MaxPage` = 500, as today).
- Lists that grow without bound (patients, payments, messages, audit): `view.Page{First: 0, Size: 20, More: true}`
  (nothing until a search) or `view.Page{First: 50, Size: 50, More: true}`. The plan of each repo says which.

## 3. A hand-written Lister

```go
func (l *thingLister) Page() view.Page { return view.Page{First: view.All} }

func (l *thingLister) List(args view.ListArgs, done func([]view.Record, error)) { … same body, args may be ignored when Page is All … }
```

## 4. Tests

- Replace every `p.Filter(term)` with `p.Search(term, nil)` and then read `p.Items()`.
- A test double that implements `view.Lister` gets `Page()` and the new `List(args, done)`.
- Add `Test<Area>_ListAcceptsListArgs`: call the list operation with `&view.ListArgs{Limit: 1}` → status 200 and
  exactly 1 row; with `&view.ListArgs{Search: "<label of one seeded row>"}` → that row; with
  `&view.ListArgs{After: <id of the first row>}` → the first row is not in the result.


## Last stage — documentation

README and `docs/ARCHITECTURE.md`: where they describe a list, say it loads as its `view.Page` declares and searches
on the server. 

## Delivery check

```
gotest
langc sync .                                   # if the repo has lang.json: 0 untranslated, 0 reuse warnings
grep -rn "\.Filter(" --include=*.go .           # → empty
grep -rn "List(done func" --include=*.go .     # → empty
```

A PR that does not compile, or whose tests could not run, is a failed delivery. Never edit a generated file
(`model_orm.go`). Commit after every stage; never run `git reset`, `git restore`, `git checkout -- <file>` or
`git stash` on your own work. Stuck for 30 minutes: open the PR anyway listing what fails.
