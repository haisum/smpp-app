package users

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
)

type permissionsRequest struct {
	URL string
}

type permissionsResponse struct {
	Permissions []permission.Permission
}

// PermissionsHandler gives list of all possible permissions a user may have
var PermissionsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := permissionsResponse{}
	uResp.Permissions = permission.GetList()
	uReq := permissionsRequest{
		URL: r.URL.RequestURI(),
	}
	resp := routes.ClientResponse{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	resp.Send(w, *r, http.StatusOK)
})
