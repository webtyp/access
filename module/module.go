package module

import "webtyp.com/appkit"

// Module is access as the entry module for appkit: users, roles, first administrator,
// and the sign-in / setup screens before the menu.
//
// Example: Modules: []appkit.Module{access.Module, catalog.Module}
var Module = appkit.Module{Name: "access", Server: server, Browser: browser}
