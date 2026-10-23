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
