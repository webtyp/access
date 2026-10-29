# webtyp/access
<img src="docs/img/badges.svg">

Every webtyp app that has users needs the same administration: create roles, give a role to a person (by e-mail), take it away, make sure the first administrators exist, and keep an audit trail. `webtyp/auth` knows users and `webtyp/rbac` knows roles, but by design they never import each other, so the piece that **crosses** them has a home here. Apps only **configure** it and mount it.

## Quick start

```go
authMod, _ := authority.New(db, auth.Config{IDs: ids})
rbacSvc, _ := rbac.New(db)

// Mount the domain layer
accessMod, _ := access.New(access.Deps{
	DB:        db,
	IDs:       ids,
	Users:     authMod,
	Roles:     rbacSvc,
	ProjectID: "my-app",
	Assurance: model.AssuranceSingleFactor,
})

// Create the tables it needs
_ = migrate.Migrate(db.RawConn(), compiler)

// Make sure the first admins exist
_ = accessMod.EnsureRole("admin", "Admin", "", []string{"boss@my-app.com"})

// Register its operations next to your app's
accessMod.MountOperations(operationRegistry)

// Mount the UI modules on the browser navigation rail
screens := ui.Screens(caller)
```

## I want X → use Y

| I want to... | Use... |
|---|---|
| create the first admins | `m.EnsureRole("admin", "Admin", "", adminEmails)` |
| let an admin manage roles | mount `m.MountOperations` and grant `access.Resources()` to the admin role |
| mount the admin screens | `screens := ui.Screens(caller)` in your UI menu registry |
| record an app action | `m.Record(actorUserID, action, target, detail)` |

## Operations

| Operation | Resource | Action | Wire Arguments |
|---|---|---|---|
| `access.list_roles` | `access_role` | `read` | (None) |
| `access.save_roles` | `access_role` | `create\|update` | `{"records":[...]}` |
| `access.delete_roles` | `access_role` | `delete` | `{"ids":[...]}` |
| `access.list_assignments` | `access_assignment` | `read` | (None) |
| `access.assign_roles` | `access_assignment` | `create` | `{"records":[...]}` |
| `access.revoke_assignments` | `access_assignment` | `delete` | `{"ids":[...]}` |
| `access.list_audit` | `access_audit` | `read` | (None) |

For architecture details, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
