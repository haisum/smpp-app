package campaign

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/user"
	log "github.com/Sirupsen/logrus"
)

type stopRequest struct {
	CampaignID string
	URL        string
	Token      string
}

type stopResponse struct {
	Count int
}

//StopHandler accepts post request to stop all pending messages of a campaign
var StopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := stopResponse{}
	var uReq stopRequest
	err := routes.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing campaign request.")
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
	if _, ok := routes.Authenticate(w, *r, uReq, uReq.Token, user.PermStopCampaign); !ok {
		return
	}
	count, err := models.StopPendingMessages(uReq.CampaignID)
	if err != nil {
		log.WithError(err).Error("Error stopping campaign.")
		resp := routes.Response{}
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't update campaign.",
			},
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	uResp.Count = count
	resp := routes.Response{
		Obj:     uResp,
		Ok:      true,
		Request: uReq,
	}
	resp.Send(w, *r, http.StatusOK)
})
