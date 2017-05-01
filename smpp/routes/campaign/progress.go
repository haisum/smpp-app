package campaign

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/campaign"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type progressRequest struct {
	CampaignID int64
	URL        string
	Token      string
}

type progressResponse struct {
	campaign.Progress
}

//ReportHandler echoes throughput report for a campaign
var ProgressHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := progressResponse{}
	var uReq progressRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing campaign progress request.")
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
	cp, err := campaign.List(campaign.Criteria{ID: uReq.CampaignID})
	var p campaign.Progress
	if err == nil && len(cp) > 0 {
		p, err = cp[0].GetProgress()
	} else {
		log.WithError(err).Error("Error getting campaign progress.")
		resp := routes.Response{}
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't get campaign progress.",
			},
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	uResp.Progress = p
	resp := routes.Response{
		Obj:     uResp,
		Ok:      true,
		Request: uReq,
	}
	resp.Send(w, *r, http.StatusOK)
})
