← [Stage 7](PLAN_STAGE_7_tests.md) | Next → [Index](PLAN.md)

> Read AGENTS.md and the index `PLAN.md` (rules, tables, external calls) first. Commit when this stage compiles.

## Stage 8 — translations and documentation

`langc sync .` (zero untranslated, zero reuse warnings). `README.md` (4 mandatory sections, ≤150 lines): operations table
(entities `access_role` read/create/update/delete, `access_assignment` read/create/delete, `access_person` read, `access_audit` read), `access.Infra` in the app example, the
note that the people list needs `access_person`. `docs/ARCHITECTURE.md`: why roles and people have their own repositories (D21) and
why they save one record at a time.
