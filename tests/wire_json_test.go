//go:build !wasm

package tests

import (
	stdjson "encoding/json"
	"testing"

	"webtyp.com/access"
	"webtyp.com/json"
	"webtyp.com/model"
)

// What a screen sends over HTTP must be JSON a server can decode: the loopback
// caller the other tests use never encodes, so a list left open shipped.
func TestWireArgs_EncodeValidJSON(t *testing.T) {
	for name, args := range map[string]model.Encodable{
		"SaveRolesArgs":   &access.SaveRolesArgs{Records: []access.RoleRecord{{}}},
		"AssignRolesArgs": &access.AssignRolesArgs{Records: []access.Assignment{{}}},
		"IDsArgs":         &access.IDsArgs{Ids: []string{"a", "b"}},
	} {
		var body []byte
		if err := json.Encode(args, &body); err != nil {
			t.Fatalf("%s: encode: %v", name, err)
		}
		if !stdjson.Valid(body) {
			t.Errorf("%s sends invalid JSON: %s", name, body)
		}
	}
}
