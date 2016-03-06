package models

// Permission represents access of a user to an operation
type Permission string

const (
	PermAddUser     Permission = "AddUser"
	PermSuspendUser            = "SuspendUser"
)

// GetPermissions returns all valid permissions for a user
func GetPermissions() []Permission {
	return []Permission{PermAddUser, PermSuspendUser}
}
