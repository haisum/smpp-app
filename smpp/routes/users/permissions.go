package users

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/routes"
	"net/http"
)

type permissionsRequest struct {
	Url string
}

type permissionsResponse struct {
	Permissions []smpp.Permission
}

// PermissionsHandler gives list of all possible permissions a user may have
var PermissionsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	uResp := permissionsResponse{}
	uResp.Permissions = smpp.GetPermissions()
	uReq := permissionsRequest{
		Url: r.URL.RequestURI(),
	}
	resp := routes.Response{
		Obj:     uResp,
		Request: uReq,
		Ok:      true,
	}
	resp.Send(w, *r, http.StatusOK)
})
