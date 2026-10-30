---
PLAN: "feat: the demo shows its API and its database (layout v0.3.99 developer tools)"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 1441580806976298907
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work.

# Developer tools in the demo

## Context

This repo's demo (`web/browser/main.go`) runs entirely in the browser: its operations go through `loopback` and its data lives
in memory, with no server. Until now the menu always added «Developer tools», and both screens asked a server that
does not exist for `/_routes` and `/_schema` (404). `webtyp.com/layout` v0.3.99 fixes that: `BuildMenu` now takes the
tools as a **required** fourth argument, and `app.LocalTools(modules, steps...)` builds them in the browser with the
same pieces the server uses — the routes the operations declare and the migration steps. So this demo stops compiling
until it passes its tools; this plan does exactly that. In a demo the explorer shows the table without the «try» form
(there is no server to send a request to).

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go get webtyp.com/layout@v0.3.99
go mod tidy
```

Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`.

## APIs used

| Symbol | Exact signature | Tag |
|---|---|---|
| `app.LocalTools` | `func LocalTools(modules []router.OperationModule, steps ...ddl.Step) (Tools, error)` | `webtyp.com/layout/app` v0.3.99 |
| `app.BuildMenu` | `func BuildMenu(screens []view.Screen, ids model.IDGenerator, allows func(model.Resource) bool, tools Tools) (Menu, error)` | v0.3.99 |
| `ddl.Step` | `struct { Name string; Run func(conn Execer, compiler Compiler) error }` | `webtyp.com/ddl` |
| each `migrate.Migrate` used below | `func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error` — fits `ddl.Step.Run` | each module |

In this file the layout package is imported as `layout_app`. If a signature differs, stop and name the symbol in
the PR; never adapt it by guessing.

## Stage 1 — `web/browser/main.go`

Add the imports (keep the existing ones; use the aliases already in the file for the root packages):

```go
	"webtyp.com/ddl"
	"webtyp.com/router"
```

Right before the existing `layout_app.BuildMenu(...)` call, build the tools from the **same modules the loopback
caller mounts** and the modules' migration steps, in this order:

```go
	steps := []ddl.Step{
		{Name: "auth", Run: authority.Migrate},
		{Name: "rbac", Run: rbac.Migrate},
		{Name: access.ModelName, Run: migrate.Migrate},
	}
	tools, err := layout_app.LocalTools([]router.OperationModule{accessMod}, steps...)
	if err != nil {
		panic(err)
	}
```

and pass `tools` as the fourth argument of the existing `BuildMenu` call (nothing else in that call changes). If `err`
is already declared in that scope, use `=` where the compiler asks for it.
Also replace the migration block at the top (`if c, ok := db.RawConn().(ddl.Compiler); ok && c != nil { _ = authority.Migrate(...); _ = rbac.Migrate(...); _ = migrate.Migrate(...) }`) so the demo runs the **same** `steps` it shows, and stops ignoring their errors:

```go
	if c, ok := db.RawConn().(ddl.Compiler); ok && c != nil {
		for _, s := range steps {
			if err := s.Run(db.RawConn(), c); err != nil {
				panic(err)
			}
		}
	}
```

Declare `steps` right after `db := orm.New(mem.New())`.

Check that it compiles for the browser: `GOOS=js GOARCH=wasm go build -o /dev/null ./web/browser`; then run `gotest`.
Commit.

## Delivery check

1. `GOOS=js GOARCH=wasm go build -o /dev/null ./web/browser` succeeds and `gotest` passes. A PR that does not compile, or whose tests could not run, is a failed delivery.
2. `grep -n "LocalTools" web/browser/main.go` → one match; every `BuildMenu(` call has four arguments.

## Delivery rules

- Commit after every stage. Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
- Stuck for 30 minutes: open the PR anyway, listing what fails.
- Do not change anything else in the demo (screens, seeds, events).

## Stages

| # | Files | Result |
|---|---|---|
| 1 | `web/browser/main.go`, `go.mod`, `go.sum` | The demo passes its tools; explorer and diagram show its own API and tables |

