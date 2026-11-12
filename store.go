package access

import "webtyp.com/crud"
import "webtyp.com/orm"

func (m *Module) StoreRoleRecord() crud.Store { return &roleStore{d: m} }
func (m *Module) StoreAssignment() crud.Store { return &assignmentStore{d: m} }
func (m *Module) StorePerson() crud.Store { return &personStore{d: m} }
func (m *Module) StoreAuditEntry() crud.Store { return orm.GlobalStore(m.db, m.ids) }
