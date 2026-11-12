package access

import (
	"webtyp.com/input"
	"webtyp.com/model"
)



// AuditEntryModel is the audit table: who did what to which target.
var AuditEntryModel = model.Definition{
	Name: "access_audit",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), ServerOnly: true, DB: &model.FieldDB{PK: true}},
		{Name: "actor_email", Type: model.Text(), Label: "Who", Item: model.ItemLabel, ServerOnly: true},
		{Name: "action", Type: model.Text(), Label: "Action", Item: model.ItemDescription, ServerOnly: true},
		{Name: "target", Type: model.Text(), Label: "Target", Item: model.ItemDescription, ServerOnly: true},
		{Name: "detail", Type: model.Text(), Label: "Detail", OmitEmpty: true, ServerOnly: true},
		{Name: "created_at", Type: model.NanoTime(), Label: "When", ServerOnly: true},
	},
	Allow: model.Read,
	MFA: model.Read,
	Rules: model.Rules{Read: model.HaveNoBusinessRule},
}

// RoleRecordModel is a role as the roles screen shows and saves it. Not a
// table: roles live in webtyp/rbac. Id equals Code.
var RoleRecordModel = model.Definition{
	Name: "access_role",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), ServerOnly: true, DB: &model.FieldDB{PK: true}},
		{Name: "code", Type: input.Text(), Label: "Code", Help: "Short identifier the permissions refer to, e.g. reception.", NotNull: true,
			Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'_'}, Minimum: 2, Maximum: 40}, Item: model.ItemDescription},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 80}, Item: model.ItemLabel},
		{Name: "description", Type: input.Textarea(), Label: "Description", OmitEmpty: true},
		{Name: "session_ttl", Type: input.Number(), Label: "Session limit (seconds)", Help: "0 keeps the application default.", OmitEmpty: true},
		{Name: "user_count", Type: model.Int(), OmitEmpty: true, ServerOnly: true},

		{Name: "actor_id", Type: model.Text(), OmitEmpty: true, ServerOnly: true},
	},
	Allow: model.Create | model.Read | model.Update | model.Delete,
	MFA: model.Create | model.Read | model.Update | model.Delete,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, Update: model.AddMyBusinessRule, Delete: model.AddMyBusinessRule},
}

// PersonModel is a user of the app as the people screen lists it: someone a
// role can be given to. Not a table: people live in webtyp/auth.
var PersonModel = model.Definition{
	Name: "access_person",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), ServerOnly: true, DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: model.Text(), Label: "Name", Item: model.ItemLabel},
		{Name: "email", Type: model.Text(), Label: "E-mail", OmitEmpty: true, Item: model.ItemDescription},
		{Name: "roles", Type: model.Text(), Label: "Roles", OmitEmpty: true, ServerOnly: true, Item: model.ItemDescription},

	},
	Allow: model.Read,
	MFA: model.Read,
	Rules: model.Rules{Read: model.HaveNoBusinessRule},
}

// AssignmentModel is "this person has this role", as the assignments screen
// shows and saves it. Not a table: assignments live in webtyp/rbac.
var AssignmentModel = model.Definition{
	Name: "access_assignment",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), ServerOnly: true, DB: &model.FieldDB{PK: true}},
		{Name: "user_id", Type: input.Select(), Label: "Person", OmitEmpty: true, Options: &PersonModel},
		{Name: "email", Type: input.Email(), Label: "Invite by e-mail", Help: "Only for someone who has never signed in. Leave it empty when you pick a person.", OmitEmpty: true},
		{Name: "role_code", Type: input.Select(), Label: "Role", NotNull: true, Item: model.ItemDescription, Options: &RoleRecordModel},
		{Name: "name", Type: model.Text(), OmitEmpty: true, Item: model.ItemLabel},

		{Name: "actor_id", Type: model.Text(), OmitEmpty: true, ServerOnly: true},
	},
	Allow: model.Create | model.Read | model.Delete,
	MFA: model.Create | model.Read | model.Delete,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, Delete: model.AddMyBusinessRule},
}

var Definition = model.ModuleDefinition{Name: "access", Entities: []model.Definition{RoleRecordModel, AssignmentModel, PersonModel, AuditEntryModel}}
