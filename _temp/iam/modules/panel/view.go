//go:build wasm

package panel

import (
	"webtyp.com/dom"
	"webtyp.com/html"
)

// Cada vista es un componente propio: Render() arma el marcado (contenedores
// vacíos identificados por los ID* de constants.go) y OnMount() —que tinywasm/dom
// invoca DESPUÉS de inyectar el HTML en el DOM— dispara el cableado (fetch +
// forms). El cableado NO puede correr en Render(): los nodos todavía no existen.
//
// No se usa rightpanel.RightPanel aquí: su wrapper toma su id de Module.ModelName(),
// que colisiona con el <section id="{moduleID}"> que platformd ya pone alrededor
// de la vista (dom entra en pánico por id duplicado).

type projectsPanel struct{ dom.Element }

func (p *projectsPanel) Render() *dom.Element {
	return html.Div().Class("iam-panel-view").Child(
		html.H2().Text("Proyectos"),
		html.Div().ID(IDProjectsNew),
		html.Div().ID(IDProjectsList),
	)
}
func (p *projectsPanel) Init(ctx dom.Ctx) { wireProjects() }

type rolesPanel struct{ dom.Element }

func (p *rolesPanel) Render() *dom.Element {
	return html.Div().Class("iam-panel-view").Child(
		html.H2().Text("Roles"),
		html.Div().Child(
			html.Label().Text("Proyecto: "),
			html.Input("select").ID(IDRolesProject),
		),
		html.Div().ID(IDRolesNew),
		html.Div().ID(IDRolesList),
	)
}
func (p *rolesPanel) Init(ctx dom.Ctx) { wireRoles() }

type usersPanel struct{ dom.Element }

func (p *usersPanel) Render() *dom.Element {
	return html.Div().Class("iam-panel-view").Child(
		html.H2().Text("Usuarios y asignación de roles"),
		html.Div().ID(IDUsersForm),
		html.Div().ID(IDUsersList),
	)
}
func (p *usersPanel) Init(ctx dom.Ctx) { wireUsers() }

type auditPanel struct{ dom.Element }

func (p *auditPanel) Render() *dom.Element {
	return html.Div().Class("iam-panel-view").Child(
		html.H2().Text("Registro de auditoría"),
		html.Div().ID(IDAuditList),
	)
}
func (p *auditPanel) Init(ctx dom.Ctx) { wireAudit() }

func projectsView() dom.Component { return &projectsPanel{} }
func rolesView() dom.Component    { return &rolesPanel{} }
func usersView() dom.Component    { return &usersPanel{} }
func auditView() dom.Component    { return &auditPanel{} }
