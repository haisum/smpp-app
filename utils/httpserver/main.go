package main

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/influx"
	"bitbucket.org/codefreak/hsmpp/smpp/license"
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
	"github.com/spf13/viper"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var (
	version     = "undefined"
	showVersion = flag.Bool("version", false, "Show binary version number.")
)

func main() {
	go license.CheckExpiry()
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}
	log.Info("Connecting database.")
	conn, err := db.Connect(viper.GetString("MYSQL_HOST"), viper.GetInt("MYSQL_PORT"), viper.GetString("MYSQL_DBNAME"), viper.GetString("MYSQL_USER"), viper.GetString("MYSQL_PASSWORD"))
	if err != nil {
		log.WithError(err).Fatal("Couldn't setup database connection.")
	}
	defer conn.Db.Close()
	_, err = db.CheckAndCreateDB()
	if err != nil {
		log.WithError(err).Fatal("Couldn't check and create db.")
	}
	log.Info("Connecting with rabbitmq.")
	q, err := queue.ConnectRabbitMQ(viper.GetString("RABBITMQ_URL"), viper.GetString("RABBITMQ_EXCHANGE"), 1)
	if err != nil {
		log.WithField("err", err).Fatalf("Error occured in connecting to rabbitmq.")
	}
	defer q.Close()
	log.Info("Connecting to influxdb.")
	_, err = influx.Connect(viper.GetString("INFLUXDB_ADDR"), viper.GetString("INFLUXDB_USERNAME"), viper.GetString("INFLUXDB_PASSWORD"))
	if err != nil {
		log.WithError(err).Fatal("Couldn't connect to influxdb")
	}
	r := mux.NewRouter()
	r.Handle("/api/message", handlers.MethodHandler{"POST": message.MessageHandler})
	r.Handle("/api/message/filter", message.MessagesHandler)
	r.Handle("/api/campaign", handlers.MethodHandler{"POST": campaign.CampaignHandler})
	r.Handle("/api/campaign/filter", campaign.CampaignsHandler)
	r.Handle("/api/campaign/report", campaign.ReportHandler)
	r.Handle("/api/campaign/progress", campaign.ProgressHandler)
	r.Handle("/api/campaign/stop", handlers.MethodHandler{"POST": campaign.StopHandler})
	r.Handle("/api/campaign/retry", handlers.MethodHandler{"POST": campaign.RetryHandler})
	r.Handle("/api/campaign/retryQd", handlers.MethodHandler{"POST": campaign.RetryQdHandler})
	r.Handle("/api/users", users.UsersHandler)
	r.Handle("/api/users/permissions", handlers.MethodHandler{"GET": users.PermissionsHandler})
	r.Handle("/api/users/add", handlers.MethodHandler{"POST": users.AddHandler})
	r.Handle("/api/users/edit", handlers.MethodHandler{"POST": users.EditHandler})
	r.Handle("/api/user/auth", handlers.MethodHandler{"POST": user.AuthHandler})
	r.Handle("/api/user/edit", handlers.MethodHandler{"POST": user.EditHandler})
	r.Handle("/api/user/info", handlers.MethodHandler{"GET": user.InfoHandler})
	r.Handle("/api/services/config", handlers.MethodHandler{"GET": services.GetConfigHandler, "POST": services.PostConfigHandler})
	r.Handle("/api/services/status", handlers.MethodHandler{"GET": services.StatusHandler, "POST": services.StatusHandler})
	r.Handle("/api/file/upload", handlers.MethodHandler{"POST": file.UploadHandler})
	r.Handle("/api/file/delete", handlers.MethodHandler{"POST": file.DeleteHandler})
	r.Handle("/api/file/download", handlers.MethodHandler{"GET": file.DownloadHandler})
	r.Handle("/api/file/filter", file.FilterHandler)
	static := http.FileServer(http.Dir("./static/"))
	r.PathPrefix("/").Handler(static)
	log.Info("Loading message workers.")
	_, err = supervisor.Execute("reload")
	if err != nil {
		if runtime.GOOS == "windows" {
			log.Error("Couldn't executing supervisor to start workers.")
		} else {
			log.Fatal("Couldn't executing supervisor to start workers.")
		}
	}

	//Listen for termination signals from OS
	go gracefulShutdown()
	log.Infof("Listening for requests on port %d", viper.GetInt("HTTP_PORT"))
	log.Fatal(http.ListenAndServeTLS(fmt.Sprintf("%s:%d", viper.GetString("HTTP_HOST"), viper.GetInt("HTTP_PORT")), viper.GetString("HTTP_CERTFILE"), viper.GetString("HTTP_KEYFILE"), handlers.CombinedLoggingHandler(os.Stdout, r)))
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
