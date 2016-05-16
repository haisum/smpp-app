package smpp

// Permission represents access of a user to an operation
type Permission string

const (
	PermAddUsers      Permission = "Add users"
	PermEditUsers                = "Edit users"
	PermListUsers                = "List users"
	PermShowConfig               = "Show config"
	PermEditConfig               = "Edit config"
	PermSendMessage              = "Send message"
	PermListMessages             = "List messages"
	PermListNumFiles             = "List number files"
	PermDeleteNumFile            = "Delete a number file"
	PermListCampaigns            = "List campaigns"
	PermStartCampaign            = "Start a campaign"
)

// GetPermissions returns all valid permissions for a user
func GetPermissions() []Permission {
	return []Permission{
		PermAddUsers,
		PermEditUsers,
		PermListUsers,
		PermShowConfig,
		PermEditConfig,
		PermSendMessage,
		PermStartCampaign,
		PermListMessages,
		PermListNumFiles,
		PermDeleteNumFile,
		PermListCampaigns,
	}
}
