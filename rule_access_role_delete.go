package access

import "webtyp.com/crud"

// Delete es la regla de negocio de borrar access_role: el receptor es la fila guardada que se va a borrar.
// crud ya validó los ids, los límites y el permiso antes de llamarte.
func (p *RoleRecord) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	p.ActorId = a.UserID
	return crud.ApplyAndPublish(), nil
}
