package module

// FirstAdminRegistrar is what a module offers to complete the first administrator (its device, its staff record).
// Optional: with no module providing it, only the user and its role are created.
type FirstAdminRegistrar interface {
	RegisterFirstAdmin(userID, code, name, ip string) error
}
