package campaign

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
)

type campaignsRequest struct {
	models.CampaignCriteria
	URL   string
	Token string
}

type campaignsResponse struct {
	Campaigns []models.Campaign
}

// CampaignsHandler handles filter requests for campaigns
var CampaignsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := campaignsResponse{}
	var uReq campaignsRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing campaign list request.")
		resp := routes.Response{
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
		u  models.User
		ok bool
	)
	if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, ""); !ok {
		return
	}
	if u.Username != uReq.Username {
		if _, ok = routes.Authenticate(w, *r, uReq, uReq.Token, smpp.PermListCampaigns); !ok {
			return
		}
	}
	camps, err := models.GetCampaigns(uReq.CampaignCriteria)
	resp := routes.Response{}
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
