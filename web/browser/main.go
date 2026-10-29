//go:build wasm

package main

import (
	"webtyp.com/access"
	"webtyp.com/access/migrate"
	"webtyp.com/access/seed"
	"webtyp.com/access/ui"
	"webtyp.com/auth"
	"webtyp.com/auth/authority"
	"webtyp.com/ddl"
	. "webtyp.com/dom"
	layout_app "webtyp.com/layout/app"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/rbac"
	"webtyp.com/router/loopback"
	"webtyp.com/storage/mem"
	"webtyp.com/unixid"
)

type demoUser struct{}

func (demoUser) UserName() string    { return "Demo Admin" }
func (demoUser) UserAvatar() string  { return "" }
func (demoUser) UserRoles() []string { return []string{"admin"} }

func main() {
	ids, err := unixid.NewUnixID()
	if err != nil {
		panic(err)
	}

	db := orm.New(mem.New())
	if c, ok := db.RawConn().(ddl.Compiler); ok && c != nil {
		_ = authority.Migrate(db.RawConn(), c)
		_ = rbac.Migrate(db.RawConn(), c)
		_ = migrate.Migrate(db.RawConn(), c)
	}

	authMod, err := authority.New(db, auth.Config{IDs: ids})
	if err != nil {
		panic(err)
	}
	rbacSvc, err := rbac.New(db)
	if err != nil {
		panic(err)
	}

	accessMod, err := access.New(access.Deps{
		DB:        db,
		IDs:       ids,
		Users:     authMod,
		Roles:     rbacSvc,
		ProjectID: "demo-project",
		Assurance: model.AssuranceSingleFactor,
	})
	if err != nil {
		panic(err)
	}

	if err := seed.Load(accessMod); err != nil {
		panic(err)
	}

	caller := loopback.ActingAs("demo-project", "admin-1", accessMod)

	menu, err := layout_app.BuildMenu(ui.Screens(caller), ids, func(model.Resource) bool { return true })
	if err != nil {
		panic(err)
	}

	p := &platformd.Platform{
		DefaultID: ui.ID + ".roles",
		Modules:   menu.Modules,
		CanView:   menu.CanView,
		User:      demoUser{},
	}

	Render("app", p)
	select {}
}
