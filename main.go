package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/haisum/smpp-app/pkg/db"
	campaignmodel "github.com/haisum/smpp-app/pkg/db/models/campaign"
	filemodel "github.com/haisum/smpp-app/pkg/db/models/campaign/file"
	msgmodel "github.com/haisum/smpp-app/pkg/db/models/message"
	usermodel "github.com/haisum/smpp-app/pkg/db/models/user"
	"github.com/haisum/smpp-app/pkg/entities/campaign/file"
	"github.com/haisum/smpp-app/pkg/errs"
	"github.com/haisum/smpp-app/pkg/excel"
	"github.com/haisum/smpp-app/pkg/logger"
	"github.com/haisum/smpp-app/pkg/response"
	"github.com/haisum/smpp-app/pkg/services/campaign"
	filesvc "github.com/haisum/smpp-app/pkg/services/campaign/file"
	"github.com/haisum/smpp-app/pkg/services/message"
	"github.com/haisum/smpp-app/pkg/services/user"
	"github.com/haisum/smpp-app/pkg/services/users"
	"github.com/haisum/smpp-app/pkg/stringutils"
)

const (
	defaultPort = "8080"
)

func main() {
	var (
		addr = envString("PORT", defaultPort)

		httpAddr        = flag.String("http.addr", ":"+addr, "HTTP listen address")
		ctx             = context.Background()
		userSvc         user.Service
		usersSvc        users.Service
		msgSvc          message.Service
		campaignSvc     campaign.Service
		campaignFileSvc filesvc.Service
	)
	flag.Parse()

	log := logger.Get()
	httpLogger := log.(logger.WithLogger).With(log, "", "component", "http")
	db := getDB(ctx, log)
	userStore := usermodel.NewStore(db, log, stringutils.Hash)
	authenticator := usermodel.NewAuthenticator(userStore.Get, stringutils.HashMatch)
	msgStore := msgmodel.NewStore(db, log)
	fileStore := filemodel.NewStore(db)
	fileOpener := file.NewOpener(envString("FILES_PATH", file.DefaultPath))
	campaignStore := campaignmodel.NewStore(db, fileStore, log)
	// user service is used for logged in user to change/access their information
	{
		userLogger := httpLogger.With("service", "user")
		userSvc = user.NewService(userLogger, userStore, authenticator)
	}
	// users service is used by privileged users to edit/add or access all the system users
	{
		usersLogger := httpLogger.With("service", "users")
		usersSvc = users.NewService(usersLogger, userStore, authenticator)
	}
	// message service is used to get reports about sent messages and sending single messages
	{
		messageLogger := httpLogger.With("service", "message")
		msgSvc = message.NewService(messageLogger, msgStore, excel.ExportMessages, authenticator)
	}
	// campaign service is used to get reports about campaigns in progress, stop campaigns and starting new campaigns
	{
		campaignLogger := httpLogger.With("service", "campaign")
		campaignSvc = campaign.NewService(campaignLogger, campaignStore, msgStore, fileStore, fileOpener, excel.ToNumbers, authenticator)
	}
	// campaign file service is used to upload, download and manage campaign files
	{
		campaignFileLogger := httpLogger.With("service", "campaign,file")
		randFunc := func() string {
			return stringutils.SecureRandomAlphaString(4) + time.Now().Format(".2006.01.02.15.04.05")
		}
		campaignFileSvc = filesvc.NewService(campaignFileLogger, fileStore, fileOpener, excel.ToNumbers, randFunc, authenticator)
	}

	mux := http.NewServeMux()

	respEncoder := response.NewEncoder(httpLogger, errs.ErrHandler, errs.ErrResponseHandler)

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(respEncoder.EncodeError),
		kithttp.ServerBefore(kithttp.PopulateRequestContext),
	}
	mux.Handle("/user/v1/", user.MakeHandler(userSvc, opts, respEncoder.EncodeSuccess))
	mux.Handle("/users/v1/", users.MakeHandler(usersSvc, opts, respEncoder.EncodeSuccess))
	mux.Handle("/message/v1/", message.MakeHandler(msgSvc, opts, respEncoder.EncodeSuccess))
	mux.Handle("/campaign/v1/", campaign.MakeHandler(campaignSvc, opts, respEncoder.EncodeSuccess))
	mux.Handle("/file/v1/", filesvc.MakeHandler(campaignFileSvc, opts, respEncoder.EncodeSuccess))
	http.Handle("/", accessControl(mux))

	errs := make(chan error, 2)
	go func() {
		log.Info("transport", "http", "address", *httpAddr, "msg", "listening")
		errs <- http.ListenAndServe(*httpAddr, nil)
	}()
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	log.Info("terminated", <-errs)

}

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}

func getDB(ctx context.Context, logger logger.Logger) *db.DB {
	db, err := db.Connect(ctx, "localhost", 3306, "hsmppdb", "root", "str0ng")
	if err != nil {
		logger.Error("error", err, "retryInSeconds", 5)
		time.Sleep(time.Second * 5)
		return getDB(ctx, logger)
	}
	return db
}

func accessControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type")

		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}
