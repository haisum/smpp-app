package campaign

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/pkg/db/models/campaign"
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/pkg/services"
	log "github.com/Sirupsen/logrus"
)

//ReportHandler echoes throughput report for a campaign
var ReportHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := reportResponse{}
	var uReq reportRequest
	err := services.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing campaign report request.")
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
	c, err := campaign.List(campaign.Criteria{ID: uReq.CampaignID})
	var cr campaign.Report
	if len(c) > 0 && err == nil {
		cr, err = c[0].GetReport()
	}
	if err != nil {
		log.WithError(err).Error("Error getting campaign report.")
		resp := services.ClientResponse{}
		resp.Errors = []services.ResponseError{
			{
				Type:    services.ErrorTypeDB,
				Message: "Couldn't get campaign report.",
			},
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	uResp.Report = cr
	resp := services.ClientResponse{
		Obj:     uResp,
		Ok:      true,
		Request: uReq,
	}
	resp.Send(w, *r, http.StatusOK)
})
