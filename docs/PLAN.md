---
PLAN: "feat: access — roles, role assignments, admin bootstrap by e-mail and audit as one reusable module over auth and rbac"
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.

# PLAN — `webtyp/access`, the access-administration module (A1a: domain + operations)

## Why

Every webtyp app that has users needs the same administration: create roles, give a role to a person
(by e-mail), take it away, make sure the first administrators exist, and keep an audit trail.
`webtyp/auth` knows users and `webtyp/rbac` knows roles, but by design they never import each other,
so the piece that **crosses** them had no home: `github.com/veltylabs/iam` wrote it inside its own
app (`config/admin_handlers.go`, `admin_backend.go`, `bootstrap.go`, `audit.go`, `panelapi.go`), and
`veltylabs/mjosefa-cms` would have had to write it again. This repo is that piece. Apps only
**configure** it (`Deps`) and mount it.

This plan builds the **domain and the operations** (no screens: those are the next plan, built on
these operations with `crudview`). The iam source is in `_temp/iam/` (same paths as in iam) as
reference: **port its behaviour, not its shape** — iam exposes hand-written REST routes and wires the
DOM by hand; here everything is a `router.OperationModule` whose wire shapes are the ones `view`/
`crudview` already speak (list with no args, save `{"records":[…]}`, delete `{"ids":[…]}`).

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/ormc/cmd/ormc@latest
go install webtyp.com/lang/cmd/langc@latest
go get webtyp.com/rbac@v0.0.21
go get webtyp.com/auth@v0.0.79
go get webtyp.com/storage@latest
```

`webtyp.com/rbac@v0.0.21` is the first tag with `(*rbac.Service).ListRoles`. If `go get` fails
because that tag lacks it, stop and say so in the PR. After adding imports run `go mod tidy`.

## Ecosystem search

- `webtyp/auth`: `auth.User`; `authority.Module` implements `UserByID`, `UserByEmail`, `CreateUser`
  (reuse through the narrow interface `Users`). `auth.NewView` is a users screen (not used here).
- `webtyp/rbac` v0.0.21: `ListRoles`, `GetRoleByCode`, `CreateRole` (upsert by id), `SetRoleSessionTTL`,
  `DeleteRoleByCode`, `AssignRoleByCode`/`RevokeRoleByCode` (idempotent), `UsersInRole`,
  `RoleUserCount`, `IsRoleNotFound` (reuse through the narrow interface `Roles`).
- `webtyp/view` (`CallerLister`): the wire shapes this module's operations must speak.
- `veltylabs/iam` (`_temp/iam/`): the behaviour to port; it stays there until iam adopts this module.
- Nothing else in `webtyp/*` or `veltylabs/*` composes users with roles → **new** repo (created).

## Design gate

1. **Prior art.** Keycloak admin console (realm roles, role mappings per user, admin events),
   Django admin (`Group` + `user.groups` + `LogEntry`), Auth0 Management API (`/roles`,
   `/users/{id}/roles`, logs). All separate *who exists* (users) from *what they may do* (roles) and
   put one administration layer over both, with an audit log. Same here: `auth` = users, `rbac` =
   roles, `access` = the administration layer. Unlike Keycloak there is no realm concept: one
   `Module` serves one rbac project, chosen by the app.
2. **Novice-name test.** `access.New(access.Deps{…})`, `m.EnsureRole(code, name, description,
   emails)` ("make sure this role exists and these people have it"), `m.Record(actorUserID, action,
   target, detail)`, operations `access.list_roles`, `access.assign_roles`, `access.list_audit`.
3. **Complexity ledger.** Concepts +1 (the module) · files an app touches: 1 (its composition root:
   `New` + mount) instead of ~1,000 lines per app · ways to administer roles: +1 / −1 once iam adopts it.
4. **Where it belongs.** The only place allowed to know both `auth` and `rbac`. Talks to them through
   two narrow interfaces declared here (`Users`, `Roles`): the concrete types are wired by the app.
5. **What it deletes.** Nothing in this repo; it enables deleting iam's `admin_handlers.go`,
   `admin_backend.go`, `bootstrap.go`, `audit.go` and part of `panelapi.go` (iam's own later plan).

## APIs used

| Symbol | Exact signature | Tag |
|---|---|---|
| `rbac.Role` | `struct { ProjectId, Id, Code, Name, Description string; SessionTtl int64 }` | rbac v0.0.21 |
| `(*rbac.Service).ListRoles` | `func (m *Service) ListRoles(projectID string) ([]Role, error)` (ordered by code) | rbac v0.0.21 |
| `(*rbac.Service).GetRoleByCode` | `func (m *Service) GetRoleByCode(projectID string, code model.RoleCode) (*Role, error)`; missing → `rbac.ErrRoleNotFound` | rbac v0.0.21 |
| `(*rbac.Service).CreateRole` | `func (m *Service) CreateRole(projectID, id string, code model.RoleCode, name, description string) error` — an existing `id` updates code/name/description | rbac v0.0.21 |
| `(*rbac.Service).SetRoleSessionTTL` | `func (m *Service) SetRoleSessionTTL(projectID, id string, ttl int64) error` | rbac v0.0.21 |
| `(*rbac.Service).DeleteRoleByCode` | `func (m *Service) DeleteRoleByCode(projectID string, code model.RoleCode) error` | rbac v0.0.21 |
| `(*rbac.Service).AssignRoleByCode` / `RevokeRoleByCode` | `(projectID, userID string, code model.RoleCode) error` — idempotent; unknown code → `ErrRoleNotFound` | rbac v0.0.21 |
| `(*rbac.Service).UsersInRole` | `func (m *Service) UsersInRole(projectID string, code model.RoleCode) ([]string, error)` | rbac v0.0.21 |
| `(*rbac.Service).RoleUserCount` | `func (m *Service) RoleUserCount(projectID string, code model.RoleCode) (int64, error)` | rbac v0.0.21 |
| `rbac.IsRoleNotFound` | `func IsRoleNotFound(err error) bool` | rbac v0.0.21 |
| `rbac.New`, `rbac.Migrate` | `func New(db *orm.DB) (*Service, error)`; `func Migrate(conn ddl.Execer, c ddl.Compiler) error` (tests) | rbac v0.0.21 |
| `auth.User` | `struct { Id, Email, Name, Phone, Status, Avatar string; CreatedAt time.Unix }` | auth v0.0.79 |
| `authority.New`, `authority.Migrate` | `func New(db *orm.DB, cfg auth.Config) (*Module, error)`; `Migrate(conn, c) error` (tests only) | auth v0.0.79 |
| `(*authority.Module)` | `UserByID(id) (auth.User, error)`, `UserByEmail(email) (auth.User, error)`, `CreateUser(email, name, phone) (auth.User, error)` | auth v0.0.79 |
| `router.OperationRegistry` | `Operation(name string, h HandlerFunc) Route` | router (via tidy) |
| `router.Route` | `Requires(model.Resource, model.Action) Route`, `RequiresMFA() Route`, `Accepts(model.Fielder) Route`, `Describe(string) Route` | router |
| `router.Context` | `UserID() string`, `Decode(model.Decodable) error`, `Encode(model.Encodable) error`, `WriteStatus(int)`, `Write([]byte)` | router |
| `model.Assurance` | `AssuranceNone` (zero), `AssuranceSingleFactor`, `AssuranceMultiFactor` | model v0.2.6 |
| `model.Fielder` | `Schema() []Field; Pointers() []any` | model v0.2.6 |
| `loopback.ActingAs` | `func ActingAs(tenantID, userID string, mods ...router.OperationModule) router.Caller` (no RBAC; tests) | `webtyp.com/router/loopback` |
| `mem.New` | `webtyp.com/storage/mem` in-memory store for `orm.New` (tests) | storage |
| `time.NowNano` | instant of an event (`time.Nano`) | `webtyp.com/time` |

If a signature in the published tag differs from this table, stop and name the symbol in the PR;
never adapt it by guessing.

## Stage 1 — `model.go` + `ormc`

```go
package access

import (
	"webtyp.com/input"
	"webtyp.com/model"
)

// ModelName is this module's identity: its operations are "access.<op>".
const ModelName = "access"

// AssignmentIDSeparator joins a role code and a user id into Assignment.Id
// ("reception/u123"): the id the assignments screen deletes by.
const AssignmentIDSeparator = "/"

// AuditEntryModel is the audit table: who did what to which target.
var AuditEntryModel = model.Definition{
	Name: "access_audit",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "actor_email", Type: model.Text(), Label: "Who"},
		{Name: "action", Type: model.Text(), Label: "Action"},
		{Name: "target", Type: model.Text(), Label: "Target"},
		{Name: "detail", Type: model.Text(), Label: "Detail", OmitEmpty: true},
		{Name: "created_at", Type: model.NanoTime(), Label: "When"},
	},
}

// RoleRecordModel is a role as the roles screen shows and saves it. Not a
// table: roles live in webtyp/rbac. Id equals Code.
var RoleRecordModel = model.Definition{
	Name: "access_role",
	Fields: model.Fields{
		{Name: "id", Type: model.Text()},
		{Name: "code", Type: input.Text(), Label: "Code", Help: "Short identifier the permissions refer to, e.g. reception.", NotNull: true,
			Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'_'}, Minimum: 2, Maximum: 40}},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 80}},
		{Name: "description", Type: input.Textarea(), Label: "Description", OmitEmpty: true},
		{Name: "session_ttl", Type: input.Number(), Label: "Session limit (seconds)", Help: "0 keeps the application default.", OmitEmpty: true},
		{Name: "user_count", Type: model.Int(), OmitEmpty: true},
	},
}

// AssignmentModel is "this person has this role", as the assignments screen
// shows and saves it. Not a table: assignments live in webtyp/rbac.
var AssignmentModel = model.Definition{
	Name: "access_assignment",
	Fields: model.Fields{
		{Name: "id", Type: model.Text()},
		{Name: "email", Type: input.Email(), Label: "E-mail", NotNull: true},
		{Name: "role_code", Type: input.Select(), Label: "Role", NotNull: true},
		{Name: "user_id", Type: model.Text(), OmitEmpty: true},
		{Name: "name", Type: model.Text(), OmitEmpty: true},
	},
}
```

Run `ormc` at the repo root. It writes `model_orm.go` with `AuditEntry`, `RoleRecord`, `Assignment`,
their `…List` types, `AuditEntry_` field names and `ReadAllAuditEntry`. Then `gofmt -w model_orm.go`,
restore `go.mod` if `ormc` touched it, `go mod tidy`. Never edit `model_orm.go` by hand; if a field of
a generated struct is named differently from what the stages below use, use the generated name.

## Stage 2 — `errors.go`

```go
package access

import (
	"webtyp.com/orm"
	"webtyp.com/router"
	"webtyp.com/rbac"
)

type domainError string

func (e domainError) Error() string { return string(e) }

const (
	ErrNoDB         domainError = "access: Deps.DB is required"
	ErrNoIDs        domainError = "access: Deps.IDs is required"
	ErrNoUsers      domainError = "access: Deps.Users is required"
	ErrNoRoles      domainError = "access: Deps.Roles is required"
	ErrNoProject    domainError = "access: Deps.ProjectID is required"
	ErrAssurance    domainError = "access: Deps.Assurance must be AssuranceSingleFactor or AssuranceMultiFactor"
	ErrNoRecords    domainError = "access: no records to save"
	ErrNoIds        domainError = "access: no ids given"
	ErrNoEmail      domainError = "access: an e-mail is required"
	ErrNoRoleCode   domainError = "access: a role code is required"
	ErrAssignmentID domainError = "access: assignment id must be <role_code>/<user_id>"
	ErrUserNotFound domainError = "access: user not found"
)

const msgDecode = "access: cannot read the request: "

// decodeError is a request body that could not be read: always 400.
type decodeError struct{ err error }

func (d decodeError) Error() string { return msgDecode + d.err.Error() }

// writeError answers err with its status and its text.
func writeError(ctx router.Context, err error) {
	ctx.WriteStatus(statusOf(err))
	ctx.Write([]byte(err.Error()))
}

func statusOf(err error) int {
	if _, ok := err.(decodeError); ok {
		return 400
	}
	if e, ok := err.(domainError); ok {
		switch e {
		case ErrUserNotFound:
			return 404
		case ErrNoRecords, ErrNoIds, ErrNoEmail, ErrNoRoleCode, ErrAssignmentID:
			return 400
		}
	}
	if rbac.IsRoleNotFound(err) || orm.IsNotFound(err) {
		return 404
	}
	return 500
}
```

## Stage 3 — `module.go`

```go
package access

import (
	"webtyp.com/auth"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/rbac"
)

// Users is what access needs from the identity store.
// *authority.Module (webtyp.com/auth/authority) implements it.
type Users interface {
	UserByID(id string) (auth.User, error)
	UserByEmail(email string) (auth.User, error)
	CreateUser(email, name, phone string) (auth.User, error)
}

// Roles is what access needs from the role store. *rbac.Service implements it.
type Roles interface {
	ListRoles(projectID string) ([]rbac.Role, error)
	GetRoleByCode(projectID string, code model.RoleCode) (*rbac.Role, error)
	CreateRole(projectID, id string, code model.RoleCode, name, description string) error
	SetRoleSessionTTL(projectID, id string, ttl int64) error
	DeleteRoleByCode(projectID string, code model.RoleCode) error
	AssignRoleByCode(projectID, userID string, code model.RoleCode) error
	RevokeRoleByCode(projectID, userID string, code model.RoleCode) error
	UsersInRole(projectID string, code model.RoleCode) ([]string, error)
	RoleUserCount(projectID string, code model.RoleCode) (int64, error)
}

// Deps is everything the app wires. Every field is required.
//
// Example:
//
//	m, err := access.New(access.Deps{DB: db, IDs: ids, Users: authMod, Roles: rbacSvc,
//		ProjectID: "mjosefa-cms", Assurance: model.AssuranceSingleFactor})
type Deps struct {
	DB        *orm.DB           // where the audit table lives (create it with migrate.Migrate)
	IDs       model.IDGenerator // ids of new roles and audit rows
	Users     Users
	Roles     Roles
	ProjectID string // the rbac project this module administers
	// Assurance is the minimum proof the operations demand of the caller:
	// AssuranceSingleFactor (a session) or AssuranceMultiFactor (a passkey,
	// what veltylabs/iam requires). The zero value is refused: choose one.
	Assurance model.Assurance
}

// Module administers the roles of one project and who holds them.
type Module struct {
	db        *orm.DB
	ids       model.IDGenerator
	users     Users
	roles     Roles
	project   string
	assurance model.Assurance
}

// New validates d and builds the module.
func New(d Deps) (*Module, error) {
	switch {
	case d.DB == nil:
		return nil, ErrNoDB
	case d.IDs == nil:
		return nil, ErrNoIDs
	case d.Users == nil:
		return nil, ErrNoUsers
	case d.Roles == nil:
		return nil, ErrNoRoles
	case d.ProjectID == "":
		return nil, ErrNoProject
	case d.Assurance != model.AssuranceSingleFactor && d.Assurance != model.AssuranceMultiFactor:
		return nil, ErrAssurance
	}
	return &Module{db: d.DB, ids: d.IDs, users: d.Users, roles: d.Roles, project: d.ProjectID, assurance: d.Assurance}, nil
}

// ModelName is the module identity operations are qualified with.
func (m *Module) ModelName() string { return ModelName }

// EnsureRole makes sure the role exists (creating it with name and
// description if missing) and that every e-mail in emails holds it,
// creating the user when there is none. Idempotent: run it at every start.
// It never touches permissions: attaching permissions to the role is the
// app's policy.
//
// Example: err := m.EnsureRole("admin", "Administrator", "Runs the clinic", []string{"ana@clinica.cl"})
func (m *Module) EnsureRole(code model.RoleCode, name, description string, emails []string) error {
	if code == "" {
		return ErrNoRoleCode
	}
	if _, err := m.roles.GetRoleByCode(m.project, code); err != nil {
		if !rbac.IsRoleNotFound(err) {
			return err
		}
		if err := m.roles.CreateRole(m.project, m.ids.NewID(), code, name, description); err != nil {
			return err
		}
	}
	for _, raw := range emails {
		email := normalizeEmail(raw)
		if email == "" {
			continue
		}
		u, err := m.userByEmailOrCreate(email)
		if err != nil {
			return err
		}
		if err := m.roles.AssignRoleByCode(m.project, u.Id, code); err != nil {
			return err
		}
	}
	return nil
}

// normalizeEmail trims and lower-cases: the same person must never become
// two users because of a capital letter.
func normalizeEmail(s string) string {
	return fmt.ToLower(fmt.Convert(s).TrimSpace().String())
}

// userByEmailOrCreate returns the user with that e-mail, creating it (name =
// e-mail) when absent — the rule iam used (_temp/iam/config/bootstrap.go).
func (m *Module) userByEmailOrCreate(email string) (auth.User, error) {
	if u, err := m.users.UserByEmail(email); err == nil {
		return u, nil
	}
	return m.users.CreateUser(email, email, "")
}
```

## Stage 4 — `audit.go`

```go
package access

import (
	"webtyp.com/time"
)

// Actions recorded by this module. An app records its own actions with
// Record and its own constants.
const (
	AuditRoleSave   = "role.save"
	AuditRoleDelete = "role.delete"
	AuditAssign     = "role.assign"
	AuditRevoke     = "role.revoke"
)

// AuditListLimit is how many rows list_audit returns, newest first.
const AuditListLimit = 200

// Record writes one audit row. actorUserID is the user who acted ("" for the
// system); the row stores that user's e-mail so it stays readable after the
// user is gone.
//
// Example: err := m.Record(ctx.UserID(), "project.create", "proj_x", "Mi proyecto")
func (m *Module) Record(actorUserID, action, target, detail string) error {
	actor := ""
	if actorUserID != "" {
		if u, err := m.users.UserByID(actorUserID); err == nil {
			actor = u.Email
		}
	}
	return m.db.Create(&AuditEntry{
		Id:         m.ids.NewID(),
		ActorEmail: actor,
		Action:     action,
		Target:     target,
		Detail:     detail,
		CreatedAt:  time.NowNano(),
	})
}

// ListAudit returns the newest AuditListLimit rows, newest first.
func (m *Module) ListAudit() ([]AuditEntry, error) {
	qb := m.db.Query(&AuditEntry{}).OrderBy(AuditEntry_.CreatedAt).Desc().Limit(AuditListLimit)
	ptrs, err := ReadAllAuditEntry(qb)
	if err != nil {
		return nil, err
	}
	out := make([]AuditEntry, len(ptrs))
	for i, p := range ptrs {
		out[i] = *p
	}
	return out, nil
}
```

If `ReadAllAuditEntry` returns `[]AuditEntry` instead of `[]*AuditEntry` (read `model_orm.go`),
return it directly. `CreatedAt`'s generated type is `time.Nano`.

## Stage 5 — `crud.go` and `queries.go`

`queries.go`:

```go
package access

import (
	"webtyp.com/model"
)

// ListRoles returns every role of the project with how many users hold it.
func (m *Module) ListRoles() ([]RoleRecord, error) {
	roles, err := m.roles.ListRoles(m.project)
	if err != nil {
		return nil, err
	}
	out := make([]RoleRecord, 0, len(roles))
	for _, r := range roles {
		count, err := m.roles.RoleUserCount(m.project, model.RoleCode(r.Code))
		if err != nil {
			return nil, err
		}
		out = append(out, RoleRecord{Id: r.Code, Code: r.Code, Name: r.Name, Description: r.Description, SessionTtl: r.SessionTtl, UserCount: count})
	}
	return out, nil
}

// ListAssignments returns, role by role, every user that holds it. A user
// that no longer exists is skipped.
func (m *Module) ListAssignments() ([]Assignment, error) {
	roles, err := m.roles.ListRoles(m.project)
	if err != nil {
		return nil, err
	}
	var out []Assignment
	for _, r := range roles {
		ids, err := m.roles.UsersInRole(m.project, model.RoleCode(r.Code))
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			u, err := m.users.UserByID(id)
			if err != nil {
				continue
			}
			out = append(out, Assignment{Id: r.Code + AssignmentIDSeparator + u.Id, Email: u.Email, RoleCode: r.Code, UserId: u.Id, Name: u.Name})
		}
	}
	return out, nil
}
```

`crud.go`:

```go
package access

import (
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/rbac"
)

// SaveRole creates the role or updates the existing one with that code.
// actorUserID is recorded in the audit.
func (m *Module) SaveRole(actorUserID string, rec RoleRecord) error {
	if rec.Code == "" {
		return ErrNoRoleCode
	}
	code := model.RoleCode(rec.Code)
	id := ""
	if existing, err := m.roles.GetRoleByCode(m.project, code); err == nil {
		id = existing.Id
	} else if !rbac.IsRoleNotFound(err) {
		return err
	} else {
		id = m.ids.NewID()
	}
	if err := m.roles.CreateRole(m.project, id, code, rec.Name, rec.Description); err != nil {
		return err
	}
	if err := m.roles.SetRoleSessionTTL(m.project, id, rec.SessionTtl); err != nil {
		return err
	}
	return m.Record(actorUserID, AuditRoleSave, rec.Code, rec.Name)
}

// DeleteRole removes the role with that code.
func (m *Module) DeleteRole(actorUserID, code string) error {
	if code == "" {
		return ErrNoRoleCode
	}
	if err := m.roles.DeleteRoleByCode(m.project, model.RoleCode(code)); err != nil {
		return err
	}
	return m.Record(actorUserID, AuditRoleDelete, code, "")
}

// Assign gives roleCode to the person with that e-mail, creating the user if
// there is none (so access can be granted before the person's first login).
func (m *Module) Assign(actorUserID, email, roleCode string) (Assignment, error) {
	email = normalizeEmail(email)
	if email == "" {
		return Assignment{}, ErrNoEmail
	}
	if roleCode == "" {
		return Assignment{}, ErrNoRoleCode
	}
	u, err := m.userByEmailOrCreate(email)
	if err != nil {
		return Assignment{}, err
	}
	if err := m.roles.AssignRoleByCode(m.project, u.Id, model.RoleCode(roleCode)); err != nil {
		return Assignment{}, err
	}
	if err := m.Record(actorUserID, AuditAssign, roleCode, email); err != nil {
		return Assignment{}, err
	}
	return Assignment{Id: roleCode + AssignmentIDSeparator + u.Id, Email: u.Email, RoleCode: roleCode, UserId: u.Id, Name: u.Name}, nil
}

// Revoke removes an assignment by its id "<role_code>/<user_id>".
func (m *Module) Revoke(actorUserID, assignmentID string) error {
	parts := fmt.Split(assignmentID, AssignmentIDSeparator)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ErrAssignmentID
	}
	u, err := m.users.UserByID(parts[1])
	if err != nil {
		return ErrUserNotFound
	}
	if err := m.roles.RevokeRoleByCode(m.project, u.Id, model.RoleCode(parts[0])); err != nil {
		return err
	}
	return m.Record(actorUserID, AuditRevoke, parts[0], u.Email)
}
```

## Stage 6 — `ops.go`

```go
package access

import (
	"webtyp.com/model"
	"webtyp.com/router"
)

// Operation names (qualified on the wire as "access.<op>").
const (
	OpListRoles         = "list_roles"
	OpSaveRoles         = "save_roles"
	OpDeleteRoles       = "delete_roles"
	OpListAssignments   = "list_assignments"
	OpAssignRoles       = "assign_roles"
	OpRevokeAssignments = "revoke_assignments"
	OpListAudit         = "list_audit"
)

// RBAC resources the operations require. The app grants them to its
// administrator role.
const (
	ResourceRole       model.Resource = "access_role"
	ResourceAssignment model.Resource = "access_assignment"
	ResourceAudit      model.Resource = "access_audit"
)

// Resources lists every resource above, for the app's permission list.
//
// Example: resources = append(resources, access.Resources()...)
func Resources() []model.Resource {
	return []model.Resource{ResourceRole, ResourceAssignment, ResourceAudit}
}

// MountOperations registers the seven operations. With AssuranceMultiFactor
// every one also RequiresMFA.
func (m *Module) MountOperations(reg router.OperationRegistry) {
	m.gate(reg.Operation(OpListRoles, m.opListRoles).Requires(ResourceRole, model.Read))
	m.gate(reg.Operation(OpSaveRoles, m.opSaveRoles).Requires(ResourceRole, model.Create|model.Update).Accepts(&SaveRolesArgs{}))
	m.gate(reg.Operation(OpDeleteRoles, m.opDeleteRoles).Requires(ResourceRole, model.Delete).Accepts(&IDsArgs{}))
	m.gate(reg.Operation(OpListAssignments, m.opListAssignments).Requires(ResourceAssignment, model.Read))
	m.gate(reg.Operation(OpAssignRoles, m.opAssignRoles).Requires(ResourceAssignment, model.Create).Accepts(&AssignRolesArgs{}))
	m.gate(reg.Operation(OpRevokeAssignments, m.opRevokeAssignments).Requires(ResourceAssignment, model.Delete).Accepts(&IDsArgs{}))
	m.gate(reg.Operation(OpListAudit, m.opListAudit).Requires(ResourceAudit, model.Read))
}

func (m *Module) gate(r router.Route) {
	if m.assurance == model.AssuranceMultiFactor {
		r.RequiresMFA()
	}
}
```

Wire args, hand-written in `ops.go` exactly like `github.com/veltylabs/service_catalog/ops.go`
(`UpsertSpecialtyArgs`, `DeleteSpecialtiesArgs`): each type has `ModelName()`, `Schema() []model.Field
{ return nil }`, `Pointers() []any { return nil }`, `IsNil()`, `EncodeFields`, `DecodeFields`.

| Type | Field | Wire | ModelName |
|---|---|---|---|
| `SaveRolesArgs` | `Records []RoleRecord` | `{"records":[…]}` (`arr.Object(&a.Records[i])`) | `"save_roles_args"` |
| `AssignRolesArgs` | `Records []Assignment` | `{"records":[…]}` | `"assign_roles_args"` |
| `IDsArgs` | `Ids []string` | `{"ids":[…]}` (`arr.String(id)`) | `"ids_args"` |

Handlers (each decodes, calls the method of Stages 4–5 with `ctx.UserID()` as actor, and answers with
`writeError` on error):

| Op | Body | Success answer |
|---|---|---|
| `opListRoles` | `m.ListRoles()` → `RoleRecordList` of pointers | `ctx.Encode(&list)` |
| `opSaveRoles` | `len(Records)==0` → `ErrNoRecords`; `SaveRole` per record | `ctx.WriteStatus(200)` |
| `opDeleteRoles` | `len(Ids)==0` → `ErrNoIds`; `DeleteRole` per id (id = code) | 200 |
| `opListAssignments` | `m.ListAssignments()` → `AssignmentList` | `ctx.Encode(&list)` |
| `opAssignRoles` | `len(Records)==0` → `ErrNoRecords`; `Assign(actor, rec.Email, rec.RoleCode)` per record | `ctx.Encode(&last)` (the last `Assignment`) |
| `opRevokeAssignments` | `len(Ids)==0` → `ErrNoIds`; `Revoke` per id | 200 |
| `opListAudit` | `m.ListAudit()` → `AuditEntryList` | `ctx.Encode(&list)` |

A decode failure is `writeError(ctx, decodeError{err})`. List operations take **no** arguments (the
project is fixed by `Deps`): do not decode a body in them.

## Stage 7 — `migrate/migrate.go` (new package `migrate`)

```go
// Package migrate creates the table access owns (the audit log). Run it at
// deploy time, next to auth's and rbac's own Migrate.
package migrate

import (
	"webtyp.com/access"
	"webtyp.com/ddl"
	"webtyp.com/model"
)

// Migrate creates access_audit if it does not exist.
//
// Example: err := migrate.Migrate(conn, compiler)
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	sorted, err := ddl.TopologicalSort([]model.Model{&access.AuditEntry{}})
	if err != nil {
		return err
	}
	return ddl.New(conn, ddlCompiler).Sync(sorted...)
}
```

## Stage 8 — tests (`tests/`, `package tests`, external)

`tests/setup_test.go`:
- `newEnv(t, assurance model.Assurance) (*access.Module, *authority.Module, *rbac.Service)`:
  `db := orm.New(mem.New())`; run `authority.Migrate`, `rbac.Migrate`, `migrate.Migrate` with
  `db.RawConn()` and `db.RawConn().(ddl.Compiler)` (as `webtyp/rbac/tests/rbac_test.go` does);
  `authMod, _ := authority.New(db, auth.Config{IDs: ids})`; `rbacSvc, _ := rbac.New(db)`;
  `access.New(access.Deps{DB: db, IDs: ids, Users: authMod, Roles: rbacSvc, ProjectID: "p1", Assurance: assurance})`.
- `seqIDs` — a `model.IDGenerator` returning `"id-1"`, `"id-2"`, …
- A recording `router.OperationRegistry` + `router.Route` (copy the shape of `mockOpRegistry`/`mockRoute`
  from `github.com/webtyp/auth` `tests/helper_test.go`) that records, per op name, the resource,
  the action and whether `RequiresMFA` was called.

| File · Test | Setup → call | Expected |
|---|---|---|
| `new_test.go` · `TestNew_RequiresEveryDep` | `access.New` with each field missing in turn, and `Assurance: model.AssuranceNone` | `ErrNoDB`, `ErrNoIDs`, `ErrNoUsers`, `ErrNoRoles`, `ErrNoProject`, `ErrAssurance` |
| `bootstrap_test.go` · `TestEnsureRole_CreatesRoleAndUsers` | `EnsureRole("admin","Admin","",["Ana@X.cl "," ","bob@x.cl"])` | role exists; users `ana@x.cl`, `bob@x.cl` exist and `UsersInRole("p1","admin")` has both |
| `bootstrap_test.go` · `TestEnsureRole_Idempotent` | call twice | still one role, two users, no error |
| `bootstrap_test.go` · `TestEnsureRole_ExistingUser` | `authMod.CreateUser("ana@x.cl","Ana","")` first | same user id holds the role (no second user) |
| `roles_test.go` · `TestSaveRole_CreateThenUpdate` | `SaveRole("", {Code:"recep",Name:"Recepción",SessionTtl:600})`, then same code `Name:"Mesón"` | `ListRoles()` one row, name `Mesón`, `SessionTtl` 600, `UserCount` 0 |
| `roles_test.go` · `TestDeleteRole` | save then `DeleteRole("", "recep")` | `ListRoles()` empty |
| `assign_test.go` · `TestAssign_CreatesUserAndLists` | role `recep`; `Assign("", "luz@x.cl", "recep")` | returned `Id == "recep/"+userId`; `ListAssignments()` one row with that e-mail |
| `assign_test.go` · `TestAssign_UnknownRole` | `Assign("", "luz@x.cl", "nope")` | `rbac.IsRoleNotFound(err)` |
| `assign_test.go` · `TestRevoke` | assign, then `Revoke("", a.Id)` | `ListAssignments()` empty |
| `assign_test.go` · `TestRevoke_BadID` | `Revoke("", "only-one-part")` | `ErrAssignmentID` |
| `audit_test.go` · `TestAudit_RecordsActorEmail` | admin user `adm@x.cl`; `SaveRole(adm.Id, …)`, `Assign(adm.Id, …)` | `ListAudit()` newest first: actions `role.assign`, `role.save`; `ActorEmail == "adm@x.cl"` |
| `ops_test.go` · `TestOps_ThroughCaller` | `caller := loopback.ActingAs("p1", adm.Id, m)`; call `access.save_roles` with `SaveRolesArgs`, `access.assign_roles`, `access.list_assignments`, `access.revoke_assignments` with `IDsArgs`, `access.list_audit` | each call's callback error is nil; the lists decode into `RoleRecordList`/`AssignmentList`/`AuditEntryList` with the expected rows |
| `ops_test.go` · `TestOps_EmptyBatch` | `access.save_roles` with no records | error (status 400) |
| `ops_test.go` · `TestOps_Gates` | `MountOperations` on the recording registry, once with `AssuranceSingleFactor`, once with `AssuranceMultiFactor` | 7 ops; resources/actions as in Stage 6; `RequiresMFA` on all 7 only in the second case |

## Stage 9 — docs and demo-free README

- `README.md` (replace the gonew stub): what this is (the paragraph of "Why", in plain words), a
  "Quick start" with the composition-root snippet (`authority.New` + `rbac.New` + `access.New` +
  `migrate.Migrate` + `m.EnsureRole("admin", …, adminEmails)` + mounting `m` with the app's other
  operation modules), a table **"I want X → use Y"** (create the first admins → `EnsureRole`; let an
  admin manage roles → mount the module and grant `access.Resources()`; record an app action →
  `Record`), and the operations table (op, resource, action, args).
- `docs/ARCHITECTURE.md` (new): why `access` exists (auth and rbac never meet), the two narrow
  interfaces, one module = one project, the assurance choice, the wire shapes, the audit table, and a
  Mermaid diagram (`flowchart TD`, no `subgraph`): `app → access`, `access → Users (auth)`,
  `access → Roles (rbac)`, `access → access_audit`. Link it from `README.md`.
- Delete `access.go` (the gonew stub) if it only holds a placeholder.

## Stage 10 — translations and cleanup

- Run `langc sync .` and fill every empty value it reports (all languages of `lang.Supported()` minus
  `en`): the labels of Stage 1 are UI text.
- Delete `_temp/` (`test ! -e _temp`).

## Delivery check

```
GOPRIVATE=webtyp.com/*,github.com/veltylabs/* GOFLAGS=-mod=mod gotest
```

- A PR that does not compile, or whose tests could not run, is a failed delivery.
- Never pause to ask permission or report progress: when the check passes, commit, push and open
  the PR. Stuck for 30 minutes: open the PR anyway listing what fails.
- Commit after every stage. Never run `git reset`, `git restore`, `git checkout -- <file>` or
  `git stash` on your own work. Never edit `model_orm.go` by hand.
- `git diff --stat main` lists only files named in the stages, `go.mod`, `go.sum`, `lang.json`,
  `AGENTS.md`, this plan, and the deletion of `_temp/` and `access.go`.

## Stages

| # | Files | Done when |
|---|---|---|
| 1 | `model.go`, `model_orm.go` (generated) | `ormc` ran; compiles |
| 2 | `errors.go` | compiles |
| 3 | `module.go` | compiles |
| 4 | `audit.go` | compiles |
| 5 | `queries.go`, `crud.go` | compiles |
| 6 | `ops.go` | compiles |
| 7 | `migrate/migrate.go` | compiles |
| 8 | `tests/*.go` | all tests pass |
| 9 | `README.md`, `docs/ARCHITECTURE.md` | written; `access.go` stub removed |
| 10 | `lang.json`, delete `_temp/` | `langc sync .` clean; `test ! -e _temp` |
