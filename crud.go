package access

import (
	"webtyp.com/auth"
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

// AssignUser gives roleCode to an existing user, picked from ListPeople.
// Example: a, err := m.AssignUser(actorID, person.Id, "reception")
func (m *Module) AssignUser(actorUserID, userID, roleCode string) (Assignment, error) {
	if userID == "" {
		return Assignment{}, ErrNoPerson
	}
	if roleCode == "" {
		return Assignment{}, ErrNoRoleCode
	}
	u, err := m.users.UserByID(userID)
	if err != nil {
		return Assignment{}, ErrUserNotFound
	}
	return m.assign(actorUserID, u, roleCode)
}

// AssignByEmail gives roleCode to the person with that e-mail, creating the
// user when there is none: an invitation, before their first sign-in.
// Example: a, err := m.AssignByEmail(actorID, "ana@example.com", "reception")
func (m *Module) AssignByEmail(actorUserID, email, roleCode string) (Assignment, error) {
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
	return m.assign(actorUserID, u, roleCode)
}

func (m *Module) assign(actorUserID string, u auth.User, roleCode string) (Assignment, error) {
	if err := m.roles.AssignRoleByCode(m.project, u.Id, model.RoleCode(roleCode)); err != nil {
		return Assignment{}, err
	}
	if err := m.Record(actorUserID, AuditAssign, roleCode, auditTarget(u)); err != nil {
		return Assignment{}, err
	}
	return Assignment{Id: roleCode + AssignmentIDSeparator + u.Id, Email: u.Email, RoleCode: roleCode, UserId: u.Id, Name: u.Name}, nil
}

// auditTarget names a person in the audit: the e-mail, or the name when there is none.
func auditTarget(u auth.User) string {
	if u.Email != "" {
		return u.Email
	}
	return u.Name
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
	return m.Record(actorUserID, AuditRevoke, parts[0], auditTarget(u))
}
