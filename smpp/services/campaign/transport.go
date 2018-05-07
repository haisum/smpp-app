package campaign

import "bitbucket.org/codefreak/hsmpp/smpp/entities/campaign"

type stopRequest struct {
	CampaignID int64
	URL        string
}

type stopResponse struct {
	Count int64
}

type reportRequest struct {
	CampaignID int64
	URL        string
}

type reportResponse struct {
	campaign.Report
}

type progressRequest struct {
	CampaignID int64
	URL        string
}

type progressResponse struct {
	campaign.Progress
}

type listRequest struct {
	campaign.Criteria
	URL string
}

type listResponse struct {
	Campaigns []campaign.Campaign
}

type startRequest struct {
	URL         string
	FileID      int64
	Numbers     string
	Description string
	Priority    int
	Src         string
	Msg         string
	ScheduledAt int64
	SendBefore  string
	SendAfter   string
	Mask        bool
	IsFlash     bool
}

type startResponse struct {
	ID int64
}
