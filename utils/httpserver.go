package main

import (
	"bitbucket.com/codefreak/hsmpp/smpp/routes/services"
	"bitbucket.com/codefreak/hsmpp/smpp/routes/user"
	"bitbucket.com/codefreak/hsmpp/smpp/routes/users"
	"bitbucket.com/codefreak/hsmpp/smpp/supervisor"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"net/http"
	"os"
)

func main() {
	r := mux.NewRouter()
	r.Handle("/api/users", handlers.MethodHandler{"GET": users.UsersHandler})
	r.Handle("/api/users/permissions", handlers.MethodHandler{"GET": users.PermissionsHandler})
	r.Handle("/api/users/add", handlers.MethodHandler{"POST": users.AddHandler})
	r.Handle("/api/users/edit", handlers.MethodHandler{"POST": users.EditHandler})
	r.Handle("/api/user/auth", handlers.MethodHandler{"POST": user.AuthHandler})
	r.Handle("/api/user/info", handlers.MethodHandler{"GET": user.InfoHandler})
	r.Handle("/api/services/config", handlers.MethodHandler{"GET": services.GetConfigHandler, "POST": services.PostConfigHandler})
	ui := http.FileServer(http.Dir("./ui/"))
	r.PathPrefix("/").Handler(ui)
	_, err := supervisor.Execute("reload")
	if err != nil {
		log.Fatal("Couldn't executing supervisor to start workers.")
	}
	log.Fatal(http.ListenAndServeTLS(":8443", "keys/cert.pem", "keys/server.key", handlers.CombinedLoggingHandler(os.Stdout, r)))
}
