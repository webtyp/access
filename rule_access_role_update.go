package access

import "webtyp.com/crud"

// Update es la regla de negocio de actualizar access_role: old es la fila guardada y el receptor trae solo los campos enviados (fields).
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (next *RoleRecord) Update(a crud.CurrentUser, tx crud.Tx, old *RoleRecord, fields []string, m *Module) (crud.RuleAnswer, error) {
	next.ActorId = a.UserID
	return crud.ApplyAndPublish(), nil
}
