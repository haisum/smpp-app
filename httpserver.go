package main

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/queue"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

// Response that's sent back to client when they send
// a request to /api/send
type SendResponse struct {
	Errors  []string
	Request queue.QueueItem
	File    string
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
	err = q.Init(c.AmqpUrl, "smppworker-exchange", 1)
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

		p, err := strconv.Atoi(r.FormValue("Priority"))
		if err != nil {
			resp.Errors = append(resp.Errors, "Priority not set correctly.")
		}

		dsts := make([]string, 0)

		f, h, err := r.FormFile("File")
		if err == nil {
			resp.File = h.Filename
			log.Info("Parsing file")
			defer f.Close()
			content, err := ioutil.ReadAll(f)
			if err != nil {
				log.WithFields(log.Fields{
					"f":   f,
					"h":   h,
					"err": err,
				}).Error("Couldn't read file")
				resp.Errors = append(resp.Errors, "Couldn't read file")
			} else {
				csv := strings.Split(string(content[:]), ",")
				for _, x := range csv {
					if len(x) > 15 {
						log.WithField("x", x).Error("Chunk too long to be a number. Skipping.")
					} else {
						dsts = append(dsts, strings.TrimRight(strings.TrimLeft(x, "\n \t"), "\n \t"))
					}
				}
			}
		}
		msg := r.FormValue("Msg")
		dst := r.FormValue("Dst")
		src := r.FormValue("Src")
		enc := r.FormValue("Enc")

		if dst != "" && len(dsts) == 0 {
			dsts = append(dsts, dst)
		}

		if msg == "" {
			resp.Errors = append(resp.Errors, "Message is empty.")
		}
		if len(dsts) == 0 {
			resp.Errors = append(resp.Errors, "Destination is empty or file couldn't be loaded.")
		}
		if src == "" {
			resp.Errors = append(resp.Errors, "Source is empty.")
		}
		if enc != "ucs" && enc != "latin" {
			resp.Errors = append(resp.Errors, "Encoding can either be \"latin\" or \"ucs\".")
		}

		resp.Request = queue.QueueItem{
			Msg:      msg,
			Dst:      dst,
			Src:      src,
			Enc:      enc,
			Priority: p,
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
		for _, d := range dsts {
			resp.Request.Dst = d
			rJson, err := resp.Request.ToJSON()
			if err != nil {
				log.WithFields(log.Fields{
					"resp.Request": resp.Request,
					"err":          err,
				}).Error("Error in formatting json request.")
				http.Error(w, "Internal server error. See logs for details.", http.StatusInternalServerError)
				return
			}
			key := matchKey(keys, resp.Request.Dst, noKey)
			log.WithFields(log.Fields{
				"key": key,
				"Dst": resp.Request.Dst,
			}).Info("Sending message.")
			err = q.Publish(key, rJson, queue.Priority(p))
			if err != nil {
				http.Error(w, "Internal server error occured. See http server logs for details.", http.StatusInternalServerError)
				return
			}
		}
		if resp.File != "" {
			resp.Request.Dst = ""
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

	log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%d", c.HttpsPort), "cert.pem", "server.key", nil))
}
