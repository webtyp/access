package access

import (
	"webtyp.com/orm"
	"webtyp.com/time"
)

// LinkRecord associates a user account with a domain record in module (D2).
// Idempotent: linking the same (userID, module, recordID) twice writes only once.
func (m *Module) LinkRecord(userID, module, recordID string) error {
	if userID == "" || module == "" || recordID == "" {
		return nil
	}
	existing := &RecordLink{}
	qb := m.db.Query(existing).
		Where(RecordLink_.UserId).Eq(userID).
		Where(RecordLink_.Module).Eq(module).
		Where(RecordLink_.RecordId).Eq(recordID)
	existing, err := ReadOneRecordLink(qb, existing)
	if err == nil && existing != nil && existing.Id != "" {
		return nil
	}
	if err != nil && !orm.IsNotFound(err) {
		return err
	}
	return m.db.Create(&RecordLink{
		Id:        m.ids.NewID(),
		UserId:    userID,
		Module:    module,
		RecordId:  recordID,
		CreatedAt: time.NowNano(),
	})
}

// LinkedRecords returns every record ID linked to userID in module (one account can hold multiple records).
func (m *Module) LinkedRecords(userID, module string) ([]string, error) {
	if userID == "" || module == "" {
		return nil, nil
	}
	qb := m.db.Query(&RecordLink{}).
		Where(RecordLink_.UserId).Eq(userID).
		Where(RecordLink_.Module).Eq(module).
		OrderBy(RecordLink_.CreatedAt).Asc()
	ptrs, err := ReadAllRecordLink(qb)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ptrs))
	for _, p := range ptrs {
		out = append(out, p.RecordId)
	}
	return out, nil
}
