package campaign

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/pkg/db/models/message"
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/pkg/services"
	log "github.com/Sirupsen/logrus"
)

//StopHandler accepts post request to stop all pending messages of a campaign
var StopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := stopResponse{}
	var uReq stopRequest
	err := services.ParseRequest(*r, &uReq)
	if err != nil {
		log.WithError(err).Error("Error parsing campaign request.")
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
	if _, ok := services.Authenticate(w, *r, uReq, uReq.Token, permission.StopCampaign); !ok {
		return
	}
	count, err := message.StopPending(uReq.CampaignID)
	if err != nil {
		log.WithError(err).Error("Error stopping campaign.")
		resp := services.ClientResponse{}
		resp.Errors = []services.ResponseError{
			{
				Type:    services.ErrorTypeDB,
				Message: "Couldn't update campaign.",
			},
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	uResp.Count = count
	resp := services.ClientResponse{
		Obj:     uResp,
		Ok:      true,
		Request: uReq,
	}
	resp.Send(w, *r, http.StatusOK)
})
