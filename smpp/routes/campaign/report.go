package campaign

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/campaign"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type reportRequest struct {
	CampaignID int64
	URL        string
	Token      string
}

type reportResponse struct {
	campaign.Report
}

//ReportHandler echoes throughput report for a campaign
var ReportHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := reportResponse{}
	var uReq reportRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing campaign report request.")
		resp := routes.Response{}
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeRequest,
				Message: "Couldn't parse request.",
			},
		}
		resp.Send(w, *r, http.StatusBadRequest)
		return
	}
	uReq.URL = r.URL.RequestURI()
	if _, ok := routes.Authenticate(w, *r, uReq, uReq.Token, permission.StartCampaign); !ok {
		return
	}
	c, err := campaign.List(campaign.Criteria{ID: uReq.CampaignID})
	var cr campaign.Report
	if len(c) > 0 && err == nil {
		cr, err = c[0].GetReport()
	}
	if err != nil {
		log.WithError(err).Error("Error getting campaign report.")
		resp := routes.Response{}
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't get campaign report.",
			},
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	uResp.Report = cr
	resp := routes.Response{
		Obj:     uResp,
		Ok:      true,
		Request: uReq,
	}
	resp.Send(w, *r, http.StatusOK)
})
