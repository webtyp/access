← [Stage 6](PLAN_STAGE_6_the_demo_is_seed_demo.md) | Next → [Stage 8](PLAN_STAGE_8_translations_and_documentati.md)

> Read AGENTS.md and the index `PLAN.md` (rules, tables, external calls) first. Commit when this stage compiles.

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
