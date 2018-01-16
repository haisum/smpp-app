package routes

// Authenticate is helper function that checks if token is valid and user has given permission
// If auth fails, it returns 401 if token is invalid or 403 if user doesn't have given permission
// "" in permisssion means this function will only check validity of token
/*func Authenticate(w http.ResponseWriter, r http.Request, req interface{}, ts string, p permission.Permission) (user.User, bool) {
	var u user.User
	resp := ClientResponse{
		Request: req,
	}
	t, err := token.Get(ts)
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
	u, err = user.Get(t.Username)
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
}*/
