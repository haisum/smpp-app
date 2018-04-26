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

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	usermodel "bitbucket.org/codefreak/hsmpp/smpp/db/models/user"
	"bitbucket.org/codefreak/hsmpp/smpp/errs"
	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/user"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/users"
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
	kithttp "github.com/go-kit/kit/transport/http"
)

const (
	defaultPort = "8080"
)

func main() {
	var (
		addr = envString("PORT", defaultPort)

		httpAddr = flag.String("http.addr", ":"+addr, "HTTP listen address")
		ctx      = context.Background()
		userSvc  user.Service
		usersSvc users.Service
	)
	flag.Parse()

	log := logger.Get()
	httpLogger := log.(logger.WithLogger).With(log, "", "component", "http")
	db := getDB(ctx, log)
	userStore := usermodel.NewStore(db, log, stringutils.Hash)
	// user service is...
	{
		userLogger := httpLogger.With("service", "user")
		authenticator := usermodel.NewAuthenticator(userStore.Get, stringutils.HashMatch)
		userSvc = user.NewService(userLogger, userStore, authenticator)
	}
	// users service is...
	{
		usersLogger := httpLogger.With("service", "users")
		authenticator := usermodel.NewAuthenticator(userStore.Get, stringutils.HashMatch)
		usersSvc = users.NewService(usersLogger, userStore, authenticator)
	}

	mux := http.NewServeMux()

	respEncoder := routes.NewResponseEncoder(httpLogger, errs.ErrHandler, errs.ErrResponseHandler)

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(respEncoder.EncodeError),
		kithttp.ServerBefore(kithttp.PopulateRequestContext),
	}
	mux.Handle("/user/v1/", user.MakeHandler(userSvc, opts, respEncoder.EncodeSuccess))
	mux.Handle("/users/v1/", users.MakeHandler(usersSvc, opts, respEncoder.EncodeSuccess))
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
	db, err := db.Connect(ctx, "localhost", 3306, "hsmppdb", "root", "")
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
