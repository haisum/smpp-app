package permission

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// Permission represents access of a user to an operation
type Permission string

// List is list of permissions
type List []Permission

const (
	// AddUsers permission to add users
	AddUsers = "Add users"
	// EditUsers permission to edit users
	EditUsers = "Edit users"
	// ListUsers permission to list/filter users
	ListUsers = "List users"
	// ShowConfig permission to see config
	ShowConfig = "Show config"
	// EditConfig permission to edit config
	EditConfig = "Edit config"
	// SendMessage permission to send messages
	SendMessage = "Send message"
	// ListMessages permission to list/filter messages
	ListMessages = "List messages"
	// ListNumFiles permission to list number files
	ListNumFiles = "List number files"
	// DeleteNumFile permission to mark a file deleted.
	DeleteNumFile = "Delete a number file"
	// ListCampaigns permission to list campaigns
	ListCampaigns = "List campaigns"
	// StartCampaign permission to start a campaign
	StartCampaign = "Start a campaign"
	// StopCampaign is permission to stop a running campaign
	StopCampaign = "Stop campaign"
	// RetryCampaign is permission to retry failed messages in campaign
	RetryCampaign = "Retry campaign"
	// GetStatus is permission to see status of running child processes via supervisord
	GetStatus = "Get status of services"
	// Mask is permission to mask messages
	Mask = "Mask Messages"
)

// GetList returns all valid permissions for a user
func GetList() List {
	return List{
		AddUsers,
		EditUsers,
		ListUsers,
		ShowConfig,
		EditConfig,
		SendMessage,
		StartCampaign,
		ListMessages,
		ListNumFiles,
		DeleteNumFile,
		ListCampaigns,
		StopCampaign,
		RetryCampaign,
		GetStatus,
		Mask,
	}
}

func (p *List) Scan(perms interface{}) error {
	ps := strings.Split(fmt.Sprintf("%s", perms), ",")
	for _, v := range ps {
		v = strings.TrimSpace(v)
		if v != "" {
			*p = append(*p, Permission(v))
		}
	}
	return nil
}

// String is string representation of permission list
// It displays comma separated permissions
func (p List) String() string {
	var perms []string
	for _, v := range p {
		perms = append(perms, string(v))
	}
	return strings.Join(perms, ",")
}

// Value implements driver.Valuer interface
func (p List) Value() (driver.Value, error) {
	return p.String(), nil
}

// Validate makes sure permissions in List are valid
func (p List) Validate() error {
	var invalids []string
	permList := GetList()
	for _, x := range p {
		var isValidPerm bool
		for _, y := range permList {
			if x == y {
				isValidPerm = true
			}
		}
		if !isValidPerm {
			invalids = append(invalids, string(x))
		}
	}
	if len(invalids) > 0 {
		return errors.New("one or more permissions are invalid:" + strings.Join(invalids, ","))
	}
	return nil
}
