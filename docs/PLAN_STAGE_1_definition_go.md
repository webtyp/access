← [Index](PLAN.md) | Next → [Stage 2](PLAN_STAGE_2_stores_go.md)

> Read AGENTS.md and the index `PLAN.md` (rules, tables, external calls) first. Commit when this stage compiles.

## Stage 1 — `definition.go`

- `Allow` as above on the four definitions. `RoleRecord.user_count` and `Person.roles` `ServerOnly: true`; `AuditEntry` fields all `ServerOnly`.
- `Assignment.Id` is the composite wire id `model.JoinKey(role_code, user_id)` (D22; replaces `AssignmentIDSeparator`, which is deleted — `grep -rn "AssignmentIDSeparator" .` → empty).
- Add `var Definition = model.ModuleDefinition{Name: "access", Entities: []model.Definition{RoleRecordModel, AssignmentModel, PersonModel, AuditEntryModel}}` and on each model `Rules` (`AddMyBusinessRule` where the table above names a hook, `HaveNoBusinessRule` otherwise; `Person` and `AuditEntry` read: `HaveNoBusinessRule`). Delete the hand-written `ModelName` if `definitionc` writes it. Run `definitionc`: it writes `definition_gen.go` (`ModelName`, `Definition()`, `MountOperations(reg crud.Registry)`) and the `rule_<entity>_<action>.go` skeletons; keep their signatures.
