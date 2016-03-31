package models

// Permission represents access of a user to an operation
type Permission string

const (
	PermAddUsers  Permission = "Add users"
	PermEditUsers            = "Edit users"
	PermListUsers            = "List users"
)

// GetPermissions returns all valid permissions for a user
func GetPermissions() []Permission {
	return []Permission{
		PermAddUsers,
		PermEditUsers,
		PermListUsers,
	}
}
