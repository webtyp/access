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
