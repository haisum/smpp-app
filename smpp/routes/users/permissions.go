package users

import (
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

type permissionsRequest struct {
	Url string
}

type permissionsResponse struct {
	Permissions []models.Permission
}

// Permissions handler gives list of all possible permissions a user may have
func Permissions(w http.ResponseWriter, r *http.Request) {
	uResp := permissionsResponse{}
	uResp.Permissions = models.GetPermissions()
	uReq := permissionsRequest{
		Url: r.URL.RequestURI(),
	}
	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	b, cType, err := routes.MakeResponse(*r, &resp)
	if err != nil {
		log.WithError(err).Error("Couldn't send permissions")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", cType)
	fmt.Fprint(w, string(b))
}
