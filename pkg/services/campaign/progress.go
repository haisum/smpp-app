package campaign

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/pkg/db/models/campaign"
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/pkg/services"
	log "github.com/Sirupsen/logrus"
)

//ReportHandler echoes throughput report for a campaign
var ProgressHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := progressResponse{}
	var uReq progressRequest
	err := services.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing campaign progress request.")
		resp := services.ClientResponse{}
		resp.Errors = []services.ResponseError{
			{
				Type:    services.ErrorTypeRequest,
				Message: "Couldn't parse request.",
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uReq.URL = r.URL.RequestURI()
	if _, ok := services.Authenticate(w, *r, uReq, uReq.Token, permission.StartCampaign); !ok {
		return
	}
	cp, err := campaign.List(campaign.Criteria{ID: uReq.CampaignID})
	var p campaign.Progress
	if err == nil && len(cp) > 0 {
		p, err = cp[0].GetProgress()
	} else {
		log.WithError(err).Error("Error getting campaign progress.")
		resp := services.ClientResponse{}
		resp.Errors = []services.ResponseError{
			{
				Type:    services.ErrorTypeDB,
				Message: "Couldn't get campaign progress.",
			},
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	uResp.Progress = p
	resp := services.ClientResponse{
		Obj:     uResp,
		Ok:      true,
		Request: uReq,
	}
	resp.Send(w, *r, http.StatusOK)
})
