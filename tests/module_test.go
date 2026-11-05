//go:build !wasm

package tests

import (
	"testing"

	"webtyp.com/access/migrate"
	"webtyp.com/access/module"
	"webtyp.com/appkit"
	"webtyp.com/ddl"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

type dummyTrustedStore struct{}

func (dummyTrustedStore) ModelName() string                  { return "trusted_store" }
func (dummyTrustedStore) IsTrustedIP(userID, ip string) bool { return true }

func TestModuleBuild(t *testing.T) {
	db := orm.New(mem.New())
	var c ddl.Compiler
	if ddlCompiler, ok := db.RawConn().(ddl.Compiler); ok {
		c = ddlCompiler
	}
	if c != nil {
		if err := migrate.Migrate(db.RawConn(), c); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}

	trustedMod := appkit.Module{
		Name: "trusted",
		Server: appkit.ModuleServer{
			Build: func(appkit.Deps) (model.ModuleNaming, error) {
				return dummyTrustedStore{}, nil
			},
		},
	}

	app := appkit.App{
		TenantID: "test",
		Timezone: "America/Santiago",
		Modules:  []appkit.Module{trustedMod, module.Module},
	}

	d, err := appkit.BuildModules(app, db)
	if err != nil {
		t.Fatalf("BuildModules: %v", err)
	}

	entry, err := appkit.Find[appkit.Entry](d)
	if err != nil {
		t.Fatalf("Find[Entry]: %v", err)
	}
	naming, ok := entry.(model.ModuleNaming)
	if !ok || naming.ModelName() != "access_entry" {
		t.Fatalf("expected access_entry, got %v", entry)
	}
}
