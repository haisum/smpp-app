package main

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/queue"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
)

type SendResponse struct {
	Errors  []string
	Request queue.QueueItem
}

// Given a list of strings and a string,
// this function returns a list item if large string starts with list item.
// string in parameter noKey is returned if no matches could be found
func matchKey(keys []string, str string, noKey string) string {
	for _, key := range keys {
		if strings.HasPrefix(str, key) {
			return key
		}
	}
	return noKey
}

func main() {
	var c smpp.Config
	err := c.LoadFile("settings.json")
	if err != nil {
		log.Fatal("Can't continue without settings. Exiting.")
	}

	var q queue.Rabbit
	err = q.Init(c.AmqpUrl, "smppworker-exchange", 5)
	if err != nil {
		log.WithField("err", err).Fatalf("Error occured in connecting to rabbitmq.")
	}

	keys := c.GetKeys()
	noKey := c.DefaultPfx

	http.HandleFunc("/api/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
			return
		}
		var resp SendResponse
		resp.Errors = make([]string, 0)

		p, err := strconv.Atoi(r.PostFormValue("Priority"))
		if err != nil {
			resp.Errors = append(resp.Errors, "Priority not set correctly.")
		}
		msg := r.PostFormValue("Msg")
		dst := r.PostFormValue("Dst")
		src := r.PostFormValue("Src")

		resp.Request = queue.QueueItem{
			Msg:      msg,
			Dst:      dst,
			Src:      src,
			Priority: p,
		}

		if msg == "" {
			resp.Errors = append(resp.Errors, "Message is empty.")
		}
		if dst == "" {
			resp.Errors = append(resp.Errors, "Destination is empty.")
		}
		if src == "" {
			resp.Errors = append(resp.Errors, "Source is empty.")
		}
		if len(resp.Errors) > 0 {
			respJson, err := json.Marshal(resp)
			if err != nil {
				log.WithFields(log.Fields{
					"resp": resp,
					"err":  err,
				}).Error("Error in formatting json response.")
				http.Error(w, "Internal server error. See logs for details.", http.StatusInternalServerError)
				return
			}
			http.Error(w, string(respJson[:]), http.StatusBadRequest)
			return
		}
		rJson, err := resp.Request.ToJSON()
		if err != nil {
			log.WithFields(log.Fields{
				"resp.Request": resp.Request,
				"err":          err,
			}).Error("Error in formatting json request.")
			http.Error(w, "Internal server error. See logs for details.", http.StatusInternalServerError)
			return
		}
		err = q.Publish(matchKey(keys, resp.Request.Dst, noKey), rJson, queue.Priority(p))
		if err != nil {
			http.Error(w, "Internal server error occured. See http server logs for details.", http.StatusInternalServerError)
			return
		}

		b, err := json.Marshal(resp)
		if err != nil {
			log.WithFields(log.Fields{
				"resp": resp,
				"err":  err,
			}).Error("Error in unmarshalling response.")
			http.Error(w, "Internal server error occured. See http server logs for details.", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(b)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "b": b}).Error("Error in writing response.")
			http.Error(w, "Internal server error occured. See http server logs for details.", http.StatusInternalServerError)
		}
	})
	log.WithFields(log.Fields{
		"HttpPort": c.HttpsPort,
	}).Info("Listening for http requests.")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", c.HttpsPort), nil))
}
