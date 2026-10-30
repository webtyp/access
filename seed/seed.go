package seed

import (
	"webtyp.com/access"
)

// Load provisions initial demonstration roles and an assignment.
func Load(m *access.Module) error {
	actorID := "seed-system"

	if err := m.SaveRole(actorID, access.RoleRecord{
		Code:        "admin",
		Name:        "Administrator",
		Description: "Full administrative access",
	}); err != nil {
		return err
	}

	if err := m.SaveRole(actorID, access.RoleRecord{
		Code:        "reception",
		Name:        "Receptionist",
		Description: "Front desk and appointments access",
	}); err != nil {
		return err
	}

	if err := m.SaveRole(actorID, access.RoleRecord{
		Code:        "viewer",
		Name:        "Auditor / Viewer",
		Description: "Read-only inspection access",
	}); err != nil {
		return err
	}

	if _, err := m.AssignByEmail(actorID, "admin@example.com", "admin"); err != nil {
		return err
	}

	return nil
}
