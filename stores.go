package access

import (
	"webtyp.com/crud"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/rbac"
)

var ErrUnsupportedCondition = fmt.Err("access: this list filters only by search or equality")

type roleStore struct {
	d *Module
}

func (s *roleStore) ReadOne(rec model.Model, id string) error {
	r, err := s.d.roles.GetRoleByCode(s.d.project, model.RoleCode(id))
	if err != nil {
		if rbac.IsRoleNotFound(err) {
			return fmt.Err("not found")
		}
		return err
	}
	count, err := s.d.roles.RoleUserCount(s.d.project, model.RoleCode(r.Code))
	if err != nil {
		return err
	}
	target := rec.(*RoleRecord)
	target.Id = r.Code
	target.Code = r.Code
	target.Name = r.Name
	target.Description = r.Description
	target.SessionTtl = r.SessionTtl
	target.UserCount = count
	return nil
}

func (s *roleStore) ReadAll(newRow func() model.Model, query crud.Query, onRow func(model.Model)) error {
	search := ""
	for _, c := range query.Conditions {
		if c.Operator() == "LIKE" {
			search = c.Value().(string)
		} else if c.Operator() != "=" {
			return ErrUnsupportedCondition
		}
	}

	list, err := s.d.ListRoles(search, query.After, int64(query.Limit))
	if err != nil {
		return err
	}
	for i := range list {
		r := list[i]
		onRow(&r)
	}
	return nil
}

func (s *roleStore) Create(rec model.Model) error {
	r := rec.(*RoleRecord)
	id := s.NewID()
	if err := s.d.roles.CreateRole(s.d.project, id, model.RoleCode(r.Code), r.Name, r.Description); err != nil {
		return err
	}
	if err := s.d.roles.SetRoleSessionTTL(s.d.project, id, r.SessionTtl); err != nil {
		return err
	}
	return s.d.Record(r.ActorId, AuditRoleSave, r.Code, r.Name)
}

func (s *roleStore) UpdateFields(rec model.Model, id string, fields []string) error {
	r := rec.(*RoleRecord)
	for _, f := range fields {
		if f != "name" && f != "description" && f != "session_ttl" {
			return crud.ErrInvalidField
		}
	}

	existing, err := s.d.roles.GetRoleByCode(s.d.project, model.RoleCode(id))
	if err != nil {
		return err
	}

	if err := s.d.roles.CreateRole(s.d.project, existing.Id, model.RoleCode(id), r.Name, r.Description); err != nil {
		return err
	}
	if err := s.d.roles.SetRoleSessionTTL(s.d.project, existing.Id, r.SessionTtl); err != nil {
		return err
	}

	return s.d.Record(r.ActorId, AuditRoleSave, id, r.Name)
}

func (s *roleStore) Delete(rec model.Model, id string) error {
	r := rec.(*RoleRecord)
	if err := s.d.roles.DeleteRoleByCode(s.d.project, model.RoleCode(id)); err != nil {
		return err
	}

	return s.d.Record(r.ActorId, AuditRoleDelete, id, "")
}

func (s *roleStore) NewID() string {
	return s.d.ids.NewID()
}

type assignmentStore struct {
	d *Module
}

func (s *assignmentStore) ReadOne(rec model.Model, id string) error {
	parts, ok := model.SplitKey(id, 2)
	if !ok || len(parts) != 2 {
		return fmt.Err("not found")
	}
	code, userID := parts[0], parts[1]
	ids, err := s.d.roles.UsersInRole(s.d.project, model.RoleCode(code))
	if err != nil {
		return err
	}
	found := false
	for _, uid := range ids {
		if uid == userID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Err("not found")
	}

	target := rec.(*Assignment)
	target.Id = id
	if u, err := s.d.users.UserByID(userID); err == nil {
		target.UserId = u.Id
		target.Email = u.Email
		target.RoleCode = code
		target.Name = u.Name
	} else {
		target.UserId = userID
		target.RoleCode = code
	}

	return nil
}

func (s *assignmentStore) ReadAll(newRow func() model.Model, query crud.Query, onRow func(model.Model)) error {
	search := ""
	for _, c := range query.Conditions {
		if c.Operator() == "LIKE" {
			search = c.Value().(string)
		} else if c.Operator() != "=" {
			return ErrUnsupportedCondition
		}
	}

	list, err := s.d.ListAssignments(search, query.After, int64(query.Limit))
	if err != nil {
		return err
	}
	for i := range list {
		a := list[i]
		onRow(&a)
	}
	return nil
}

func (s *assignmentStore) Create(rec model.Model) error {
	r := rec.(*Assignment)
	if r.UserId != "" {
		a, err := s.d.AssignUser(r.ActorId, r.UserId, r.RoleCode)
		if err != nil {
			return err
		}
		r.Id = a.Id
		return nil
	}
	if r.Email != "" {
		a, err := s.d.AssignByEmail(r.ActorId, r.Email, r.RoleCode)
		if err != nil {
			return err
		}
		r.Id = a.Id
		return nil
	}
	return crud.ErrInvalidField
}

func (s *assignmentStore) UpdateFields(rec model.Model, id string, fields []string) error {
	return crud.ErrInvalidField
}

func (s *assignmentStore) Delete(rec model.Model, id string) error {
	r := rec.(*Assignment)
	return s.d.Revoke(r.ActorId, id)
}

func (s *assignmentStore) NewID() string {
	return s.d.ids.NewID()
}

type personStore struct {
	d *Module
}

func (s *personStore) ReadOne(rec model.Model, id string) error {
	u, err := s.d.users.UserByID(id)
	if err != nil {
		return fmt.Err("not found")
	}

	target := rec.(*Person)
	target.Id = u.Id
	target.Name = u.Name
	target.Email = u.Email

	roles, err := s.d.roles.ListRoles(s.d.project)
	if err == nil {
		for _, r := range roles {
			ids, err := s.d.roles.UsersInRole(s.d.project, model.RoleCode(r.Code))
			if err != nil {
				continue
			}
			for _, uid := range ids {
				if uid == u.Id {
					target.Roles = joinRoleName(target.Roles, r.Name)
					break
				}
			}
		}
	}
	return nil
}

func (s *personStore) ReadAll(newRow func() model.Model, query crud.Query, onRow func(model.Model)) error {
	search := ""
	for _, c := range query.Conditions {
		if c.Operator() == "LIKE" {
			search = c.Value().(string)
		} else if c.Operator() != "=" {
			return ErrUnsupportedCondition
		}
	}

	list, err := s.d.ListPeople(search, query.After, int64(query.Limit))
	if err != nil {
		return err
	}
	for i := range list {
		p := list[i]
		onRow(&p)
	}
	return nil
}

func (s *personStore) Create(rec model.Model) error {
	return crud.ErrInvalidField
}

func (s *personStore) UpdateFields(rec model.Model, id string, fields []string) error {
	return crud.ErrInvalidField
}

func (s *personStore) Delete(rec model.Model, id string) error {
	return crud.ErrInvalidField
}

func (s *personStore) NewID() string {
	return s.d.ids.NewID()
}
