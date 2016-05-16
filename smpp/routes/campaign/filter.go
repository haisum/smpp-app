package campaign

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type campaignsRequest struct {
	models.CampaignCriteria
	Url      string
	Token    string
	Username string
}

type campaignsResponse struct {
	Campaigns []models.Campaign
}

// MessagesHandler allows adding a user to database
var CampaignsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := campaignsResponse{}
	var uReq campaignsRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing campaign list request.")
		resp := routes.Response{
			Errors: routes.ResponseErrors{
				http.StatusText(http.StatusBadRequest): "Couldn't parse request",
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uReq.Url = r.URL.RequestURI()
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
		resp.Errors = routes.ResponseErrors{"db": "Couldn't get campaigns."}
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
