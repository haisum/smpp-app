package main

import (
	"bitbucket.com/codefreak/hsmpp/smpp/routes/users"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/users", users.Users).Methods("POST", "GET")
	r.HandleFunc("/api/users/permissions", users.Permissions).Methods("GET")
	r.HandleFunc("/api/users/add", users.Add).Methods("POST")
	r.HandleFunc("/api/users/edit", users.Edit).Methods("POST")
	log.Fatal(http.ListenAndServe(":8443", r))
}
