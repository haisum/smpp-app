package campaign

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/user"
	log "github.com/Sirupsen/logrus"
)

type reportRequest struct {
	CampaignID string
	URL        string
	Token      string
}

type reportResponse struct {
	models.CampaignReport
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
	if _, ok := routes.Authenticate(w, *r, uReq, uReq.Token, user.PermStartCampaign); !ok {
		return
	}
	cr, err := models.GetReport(uReq.CampaignID)
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
	uResp.CampaignReport = cr
	resp := routes.Response{
		Obj:     uResp,
		Ok:      true,
		Request: uReq,
	}
	resp.Send(w, *r, http.StatusOK)
})
