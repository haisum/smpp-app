package user

// Permission represents access of a user to an operation
type Permission string

const (
	// PermAddUsers permission to add users
	PermAddUsers Permission = "Add users"
	// PermEditUsers permission to edit users
	PermEditUsers = "Edit users"
	// PermListUsers permission to list/filter users
	PermListUsers = "List users"
	// PermShowConfig permission to see config
	PermShowConfig = "Show config"
	// PermEditConfig permission to edit config
	PermEditConfig = "Edit config"
	// PermSendMessage permission to send messages
	PermSendMessage = "Send message"
	// PermListMessages permission to list/filter messages
	PermListMessages = "List messages"
	// PermListNumFiles permission to list number files
	PermListNumFiles = "List number files"
	// PermDeleteNumFile permission to mark a numfile deleted.
	PermDeleteNumFile = "Delete a number file"
	// PermListCampaigns permission to list campaigns
	PermListCampaigns = "List campaigns"
	// PermStartCampaign permission to start a campaign
	PermStartCampaign = "Start a campaign"
	// PermStopCampaign is permission to stop a running campaign
	PermStopCampaign = "Stop campaign"
	// PermRetryCampaign is permission to retry failed messages in campaign
	PermRetryCampaign = "Retry campaign"
	// PermGetStatus is permission to see status of running child processes via supervisord
	PermGetStatus = "Get status of services"
	// PermMask is permission to mask messages
	PermMask = "Mask Messages"
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
		PermStopCampaign,
		PermRetryCampaign,
		PermGetStatus,
		PermMask,
	}
}
