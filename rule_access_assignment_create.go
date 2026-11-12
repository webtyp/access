package access

import (
	"webtyp.com/crud"
	"webtyp.com/fmt"
)

// Create es la regla de negocio de crear access_assignment: valida lo que solo tu dominio sabe y responde si se guarda y si se avisa.
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (p *Assignment) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	p.ActorId = a.UserID
	if (p.UserId == "" && p.Email == "") || (p.UserId != "" && p.Email != "") {
		return crud.RuleAnswer{}, fmt.Err("access: exactly one of user_id or email is required")
	}
	return crud.ApplyAndPublish(), nil
}
