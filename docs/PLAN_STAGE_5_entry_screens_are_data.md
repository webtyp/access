← [Stage 4](PLAN_STAGE_4_access_infra.md) | Next → [Stage 6](PLAN_STAGE_6_the_demo_is_seed_demo.md)

> Read AGENTS.md and the index `PLAN.md` (rules, tables, external calls) first. Commit when this stage compiles.

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
