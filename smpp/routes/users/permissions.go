package users

import (
	"net/http"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
)

type permissionsRequest struct {
	URL string
}

type permissionsResponse struct {
	Permissions []smpp.Permission
}

// PermissionsHandler gives list of all possible permissions a user may have
var PermissionsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := permissionsResponse{}
	uResp.Permissions = smpp.GetPermissions()
	uReq := permissionsRequest{
		URL: r.URL.RequestURI(),
	}
	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	resp.Send(w, *r, http.StatusOK)
})
