package main

import (
	"context"
	"flag"
	"os"

	"time"

	"net/http"

	"fmt"
	"os/signal"
	"syscall"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/token"
	usermodel "bitbucket.org/codefreak/hsmpp/smpp/db/models/user"
	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/user"
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
	)
	flag.Parse()

	log := logger.FromContext(ctx)
	httpLogger := log.(logger.WithLogger).With(log, "component", "http")
	db := getDB(ctx, log)
	// user service is...
	{
		tokenStore := token.NewStore(db, log)
		userStore := usermodel.NewStore(db, log)
		userLogger := httpLogger.With("service", "user")
		userSvc = user.NewService(db, userLogger, tokenStore, userStore)
	}

	mux := http.NewServeMux()

	mux.Handle("/user/v1/", user.MakeHandler(userSvc))
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
