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
