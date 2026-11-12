package access

import "webtyp.com/crud"

// Create es la regla de negocio de crear access_role: valida lo que solo tu dominio sabe y responde si se guarda y si se avisa.
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (p *RoleRecord) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	p.ActorId = a.UserID
	return crud.ApplyAndPublish(), nil
}
