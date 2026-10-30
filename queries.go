package access

import (
	"webtyp.com/model"
)

// roleNameSeparator joins the names of the roles a person holds.
const roleNameSeparator = ", "

// ListPeople returns every user of the app with the names of the roles they
// hold in this project: the list a role is given from.
// Example: people, err := m.ListPeople()
func (m *Module) ListPeople() ([]Person, error) {
	users, err := m.users.ListUsers()
	if err != nil {
		return nil, err
	}
	roles, err := m.ListRoles()
	if err != nil {
		return nil, err
	}
	assigned, err := m.ListAssignments()
	if err != nil {
		return nil, err
	}
	out := make([]Person, 0, len(users))
	for _, u := range users {
		p := Person{Id: u.Id, Name: u.Name, Email: u.Email}
		for _, a := range assigned {
			if a.UserId == u.Id {
				p.Roles = joinRoleName(p.Roles, roleName(roles, a.RoleCode))
			}
		}
		out = append(out, p)
	}
	return out, nil
}

func roleName(roles []RoleRecord, code string) string {
	for _, r := range roles {
		if r.Code == code {
			return r.Name
		}
	}
	return code
}

func joinRoleName(list, name string) string {
	if list == "" {
		return name
	}
	return list + roleNameSeparator + name
}

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
