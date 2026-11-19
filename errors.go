package access

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
	ErrNoPerson     domainError = "access: pick a person or give an e-mail"
	ErrNoRoleCode   domainError = "access: a role code is required"
	ErrAssignmentID domainError = "access: assignment id must be <role_code>/<user_id>"
	ErrUserNotFound domainError = "access: user not found"
)
