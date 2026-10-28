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
		{Name: "actor_email", Type: model.Text(), Label: "Who", Row: model.RowTitle},
		{Name: "action", Type: model.Text(), Label: "Action", Row: model.RowDetail},
		{Name: "target", Type: model.Text(), Label: "Target", Row: model.RowDetail},
		{Name: "detail", Type: model.Text(), Label: "Detail", OmitEmpty: true, Row: model.RowDetail},
		{Name: "created_at", Type: model.NanoTime(), Label: "When"},
	},
}

// RoleRecordModel is a role as the roles screen shows and saves it. Not a
// table: roles live in webtyp/rbac. Id equals Code.
var RoleRecordModel = model.Definition{
	Name: "access_role",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "code", Type: input.Text(), Label: "Code", Help: "Short identifier the permissions refer to, e.g. reception.", NotNull: true,
			Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'_'}, Minimum: 2, Maximum: 40}, Row: model.RowDetail},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 80}, Row: model.RowTitle},
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
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "email", Type: input.Email(), Label: "E-mail", NotNull: true, Row: model.RowTitle},
		{Name: "role_code", Type: input.Select(), Label: "Role", NotNull: true, Row: model.RowDetail, Options: &RoleRecordModel},
		{Name: "user_id", Type: model.Text(), OmitEmpty: true},
		{Name: "name", Type: model.Text(), OmitEmpty: true, Row: model.RowDetail},
	},
}
