package main

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/campaign"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/file"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/message"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/services"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/user"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/users"
	"bitbucket.org/codefreak/hsmpp/smpp/supervisor"
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	port    = flag.Int("port", 8443, "Port on which http service should start.")
	amqpUrl = flag.String("amqpUrl", "amqp://guest:guest@localhost:5672/", "Amqp url for rabbitmq")
)

func main() {
	flag.Parse()
	log.Info("Connecting database.")
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Fatal("Couldn't setup database connection.")
	}
	defer s.Close()

	log.Info("Connecting with rabbitmq.")
	q, err := queue.GetQueue(*amqpUrl, "smppworker-exchange", 1)
	if err != nil {
		log.WithField("err", err).Fatalf("Error occured in connecting to rabbitmq.")
	}
	defer q.Close()
	r := mux.NewRouter()
	r.Handle("/api/message", handlers.MethodHandler{"POST": message.MessageHandler})
	r.Handle("/api/message/filter", handlers.MethodHandler{"GET": message.MessagesHandler})
	r.Handle("/api/campaign", handlers.MethodHandler{"POST": campaign.CampaignHandler})
	r.Handle("/api/users", handlers.MethodHandler{"GET": users.UsersHandler})
	r.Handle("/api/users/permissions", handlers.MethodHandler{"GET": users.PermissionsHandler})
	r.Handle("/api/users/add", handlers.MethodHandler{"POST": users.AddHandler})
	r.Handle("/api/users/edit", handlers.MethodHandler{"POST": users.EditHandler})
	r.Handle("/api/user/auth", handlers.MethodHandler{"POST": user.AuthHandler})
	r.Handle("/api/user/info", handlers.MethodHandler{"GET": user.InfoHandler})
	r.Handle("/api/services/config", handlers.MethodHandler{"GET": services.GetConfigHandler, "POST": services.PostConfigHandler})
	r.Handle("/api/file/upload", handlers.MethodHandler{"POST": file.UploadHandler})
	r.Handle("/api/file/delete", handlers.MethodHandler{"POST": file.DeleteHandler})
	r.Handle("/api/file/filter", handlers.MethodHandler{"GET": file.FilterHandler})
	ui := http.FileServer(http.Dir("./ui/"))
	r.PathPrefix("/").Handler(ui)
	log.Info("Loading message workers.")
	_, err = supervisor.Execute("reload")
	if err != nil {
		log.Fatal("Couldn't executing supervisor to start workers.")
	}

	//Listen for termination signals from OS
	go gracefulShutdown()
	log.Infof("Listening for requests on port %d", *port)
	log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%d", *port), "keys/cert.pem", "keys/server.key", handlers.CombinedLoggingHandler(os.Stdout, r)))
}

// When SIGTERM or SIGINT is received, this routine will close our workers
func gracefulShutdown() {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	signal.Notify(s, syscall.SIGTERM)
	go func() {
		<-s
		log.Print("Sutting down gracefully.")
		supervisor.Execute("stop")
		os.Exit(0)
	}()
}
