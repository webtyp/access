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
