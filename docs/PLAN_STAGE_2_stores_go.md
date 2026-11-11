← [Stage 1](PLAN_STAGE_1_definition_go.md) | Next → [Stage 3](PLAN_STAGE_3_ops_go_goes_away_the_rules.md)

> Read AGENTS.md and the index `PLAN.md` (rules, tables, external calls) first. Commit when this stage compiles.

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
