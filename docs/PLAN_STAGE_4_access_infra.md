← [Stage 3](PLAN_STAGE_3_ops_go_goes_away_the_rules.md) | Next → [Stage 5](PLAN_STAGE_5_entry_screens_are_data.md)

> Read AGENTS.md and the index `PLAN.md` (rules, tables, external calls) first. Commit when this stage compiles.

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
