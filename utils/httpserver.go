package main

import (
	"bitbucket.com/codefreak/hsmpp/smpp/routes/user"
	"bitbucket.com/codefreak/hsmpp/smpp/routes/users"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"net/http"
	"os"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/api/users", users.Users).Methods("POST", "GET")
	r.HandleFunc("/api/users/permissions", users.Permissions).Methods("GET")
	r.HandleFunc("/api/users/add", users.Add).Methods("POST")
	r.HandleFunc("/api/users/edit", users.Edit).Methods("POST")
	r.HandleFunc("/api/user/auth", user.Auth).Methods("POST")
	ui := http.FileServer(http.Dir("./ui/"))
	r.PathPrefix("/").Handler(ui)
	log.Fatal(http.ListenAndServeTLS(":8443", "keys/cert.pem", "keys/server.key", handlers.CombinedLoggingHandler(os.Stdout, r)))
}
