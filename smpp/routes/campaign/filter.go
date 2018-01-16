package campaign

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models/campaign"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
)

type campaignsRequest struct {
	campaign.Criteria
	URL   string
	Token string
}

type campaignsResponse struct {
	Campaigns []campaign.Campaign
}

// CampaignsHandler handles filter requests for campaigns
var CampaignsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := campaignsResponse{}
	var uReq campaignsRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing campaign list request.")
		resp := routes.ClientResponse{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeRequest,
					Message: "Couldn't parse request.",
				},
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uReq.URL = r.URL.RequestURI()
	var (
		u  user.User
		ok bool
	)
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, ""); !ok {
		return
	}
	if u.Username != uReq.Username {
		if _, ok = routes.Authenticate(w, *r, uReq, uReq.Token, permission.ListCampaigns); !ok {
			return
		}
	}
	camps, err := campaign.List(uReq.Criteria)
	resp := routes.ClientResponse{}
	if err != nil {
		resp.Ok = false
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't get campaigns.",
			},
		}
		resp.Request = uReq
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uResp.Campaigns = camps
	resp.Obj = uResp
	resp.Ok = true
	resp.Request = uReq
	resp.Send(w, *r, http.StatusOK)
})
