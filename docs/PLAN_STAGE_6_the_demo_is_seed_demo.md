← [Stage 5](PLAN_STAGE_5_entry_screens_are_data.md) | Next → [Stage 7](PLAN_STAGE_7_tests.md)

> Read AGENTS.md and the index `PLAN.md` (rules, tables, external calls) first. Commit when this stage compiles.

## Stage 6 — the demo is `seed.Demo` (D15)

New `seed/demo.go` (`//go:build wasm`), from today's `web/browser/main.go`: the same `mem` database and three migration steps run on the
connection, the same `authority`, `rbac`, `access` modules and seed call; it returns
`view.Demo{Label: "Access", Timezone: "America/Santiago", ActingAs: <the admin id today's main signs in as>, Modules: []crud.Module{authMod, accessMod}, Migrations: steps, Screens: func(d view.BrowserDeps) []view.Screen { return ui.Screens(d.Caller) }}`.
Delete `web/browser/main.go` (`test ! -e web`). After `go mod tidy`: `grep -rn "webtyp.com/layout" .` → empty, `go.mod` included.
