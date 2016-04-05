package routes

import (
	"bitbucket.com/codefreak/hsmpp/smpp/db"
	"bitbucket.com/codefreak/hsmpp/smpp/db/models"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

func Authenticate(w http.ResponseWriter, r http.Request, req interface{}, token string, p models.Permission) bool {
	resp := Response{
		Request: req,
	}
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		resp.Errors = ResponseErrors{
			"auth": "Couldn't connect db.",
		}
		resp.Send(w, r, http.StatusInternalServerError)
		return false
	}
	t, err := models.GetToken(s, token)
	if err != nil {
		log.WithError(err).Error("Couldn't get token.")
		resp.Errors = ResponseErrors{
			"auth": "Invalid token.",
		}
		resp.Send(w, r, http.StatusUnauthorized)
		return false
	}
	u, err := models.GetUser(s, t.Username)
	if err != nil {
		log.WithError(err).Error("Couldn't get user.")
		resp.Errors = ResponseErrors{
			"auth": "This user no longer exists.",
		}
		resp.Send(w, r, http.StatusUnauthorized)
		return false
	}
	if u.Suspended {
		resp.Errors = ResponseErrors{
			"auth": "This user is suspended.",
		}
		resp.Send(w, r, http.StatusUnauthorized)
		return false
	}
	if p != "" {
		for _, perm := range u.Permissions {
			if perm == p {
				return true
			}
		}
		resp.Errors = ResponseErrors{
			"auth": "You don't have permissions to access this resource.",
		}
		resp.Send(w, r, http.StatusForbidden)
		return false
	}
	return true
}