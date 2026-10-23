package config

import "webtyp.com/model"

// Project is iam's own concept, not rbac's: rbac knows roles/permissions
// scoped by project_id, but has no notion of an application authenticating
// itself to ask for a token — that credential belongs here.
//
//go:generate ormc
var ProjectModel = model.Definition{
	Name: "project",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: model.Text()},
		{Name: "client_public_key", Type: model.Text()},
		{Name: "created_at", Type: model.UnixTime()},
		// active: 1 = activo, 0 = desactivado. Fila sin valor (previa a esta
		// migración) se trata como ACTIVA — ver 1.5. Sin widget: no se edita en form.
		{Name: "active", Type: model.Int()},
	},
}

var AuditEntryModel = model.Definition{
	Name: "audit_entry",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "actor_email", Type: model.Text()},
		{Name: "action", Type: model.Text()},
		{Name: "target", Type: model.Text()},
		{Name: "detail", Type: model.Text()},
		{Name: "created_at", Type: model.UnixTime()},
	},
}
