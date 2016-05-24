package routes

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	log "github.com/Sirupsen/logrus"
	"net/http"
)

func Authenticate(w http.ResponseWriter, r http.Request, req interface{}, token string, p smpp.Permission) (models.User, bool) {
	var u models.User
	resp := Response{
		Request: req,
	}
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		resp.Errors = []ResponseError{
			{
				Type:    ErrorTypeDB,
				Message: "Couldn't connect database.",
			},
		}
		resp.Send(w, r, http.StatusInternalServerError)
		return u, false
	}
	t, err := models.GetToken(s, token)
	if err != nil {
		log.WithError(err).Error("Couldn't get token.")
		resp.Errors = []ResponseError{
			{
				Type:    ErrorTypeAuth,
				Message: "Invalid token.",
			},
		}
		resp.Send(w, r, http.StatusUnauthorized)
		return u, false
	}
	u, err = models.GetUser(s, t.Username)
	if err != nil {
		log.WithError(err).Error("Couldn't get user.")
		resp.Errors = []ResponseError{
			{
				Type:    ErrorTypeAuth,
				Message: "This user no longer exists.",
			},
		}
		resp.Send(w, r, http.StatusUnauthorized)
		return u, false
	}
	if u.Suspended {
		resp.Errors = []ResponseError{
			{
				Type:    ErrorTypeAuth,
				Message: "This user is suspended.",
			},
		}
		resp.Send(w, r, http.StatusUnauthorized)
		return u, false
	}
	if p != "" {
		for _, perm := range u.Permissions {
			if perm == p {
				return u, true
			}
		}
		resp.Errors = []ResponseError{
			{
				Type:    ErrorTypeAuth,
				Message: "You don't have permissions to access this resource.",
			},
		}
		resp.Send(w, r, http.StatusForbidden)
		return u, false
	}
	return u, true
}
