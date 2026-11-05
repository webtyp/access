package module

import "webtyp.com/auth"

// FirstAdminRegistrar is what a module offers to complete the first administrator (its device, its staff record).
// Optional: with no module providing it, only the user and its role are created.
type FirstAdminRegistrar interface {
	RegisterFirstAdmin(userID, code, name, ip string) error
}

// StaffDevices is the staff module's register of people and their devices: it says from which IP a person may
// enter and completes the first administrator. Both from the same provider — the users module also answers
// IsTrustedIP (its own LAN table), so a contract with only that method would match two modules.
//
// Example: staff, err := appkit.Find[StaffDevices](d)
type StaffDevices interface {
	auth.TrustedIPStore
	FirstAdminRegistrar
}
