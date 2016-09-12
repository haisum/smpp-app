package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/license"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"bitbucket.org/codefreak/hsmpp/smpp/smtext"
	"bitbucket.org/codefreak/hsmpp/smpp/soap"
	log "github.com/Sirupsen/logrus"
)

const (
	//HTTPPort is port on which soapservice listens
	HTTPPort int = 8445
)

func main() {
	go license.CheckExpiry()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		decoder := xml.NewDecoder(r.Body)
		var e soap.Envelope
		err := decoder.Decode(&e)
		if err != nil {
			http.Error(w, fmt.Sprintf(soap.Response, "Couldn't understand soap request.", ""), http.StatusBadRequest)
			return
		}
		_, err = db.GetSession()
		if err != nil {
			http.Error(w, fmt.Sprintf(soap.Response, "Couldn't connect to database.", ""), http.StatusInternalServerError)
			return
		}
		u, err := models.GetUser(e.Body.Request.Username)
		if err != nil {
			http.Error(w, fmt.Sprintf(soap.Response, "Username/password is wrong.", ""), http.StatusUnauthorized)
			return
		}
		if !u.Auth(e.Body.Request.Password) {
			http.Error(w, fmt.Sprintf(soap.Response, "Username/password is wrong.", ""), http.StatusUnauthorized)
			return
		}
		q, err := queue.GetQueue("amqp://guest:guest@localhost:5672/", "smppworker-exchange", 1)
		config, err := models.GetConfig()
		keys := config.GetKeys(u.ConnectionGroup)
		var noKey string
		var group smpp.ConnGroup
		if group, err = config.GetGroup(u.ConnectionGroup); err != nil {
			http.Error(w, fmt.Sprintf(soap.Response, "User's connection group doesn't exist in configuration.", ""), http.StatusUnauthorized)
			return
		}
		enc := smtext.EncLatin
		if !smtext.IsASCII(e.Body.Request.Message) {
			enc = smtext.EncUCS
		}
		total := smtext.Total(e.Body.Request.Message, enc)

		if e.Body.Request.Priority == 0 {
			e.Body.Request.Priority = 7
		}

		m := models.Message{
			ConnectionGroup: u.ConnectionGroup,
			Username:        u.Username,
			Msg:             e.Body.Request.Message,
			RealMsg:         e.Body.Request.Message,
			Priority:        e.Body.Request.Priority,
			Enc:             enc,
			Dst:             e.Body.Request.Dst,
			Src:             e.Body.Request.Src,
			QueuedAt:        time.Now().UTC().Unix(),
			Status:          models.MsgQueued,
			Total:           total,
			SendAfter:       e.Body.Request.SendAfter,
			SendBefore:      e.Body.Request.SendBefore,
		}
		errors := m.Validate()
		if len(errors) != 0 {
			http.Error(w, fmt.Sprintf(soap.Response, strings.Join(errors, "\n"), ""), http.StatusBadRequest)
			return
		}
		msgID, err := m.Save()
		if err != nil {
			http.Error(w, fmt.Sprintf(soap.Response, "Couldn't save message.", ""), http.StatusInternalServerError)
			return
		}
		noKey = group.DefaultPfx
		key := matchKey(keys, m.Dst, noKey)
		qItem := queue.Item{
			MsgID: msgID,
			Total: total,
		}
		respJSON, _ := qItem.ToJSON()
		err = q.Publish(fmt.Sprintf("%s-%s", u.ConnectionGroup, key), respJSON, queue.Priority(m.Priority))
		if err != nil {
			log.WithError(err).Error("Error sending message.")
			fmt.Fprintf(w, soap.Response, "Error in queueing message.", "")
		} else {
			log.WithField("msg", m).Info("Sent message.")
			fmt.Fprintf(w, soap.Response, "OK", msgID)
		}
		return
	})
	http.HandleFunc("/wsdl", func(w http.ResponseWriter, r *http.Request) {
		host := r.FormValue("host")
		port := r.FormValue("port")
		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = strconv.Itoa(HTTPPort)
		}
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, soap.WSDL, host, port)
		return
	})
	log.Infof("Listening on port %s.", HTTPPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", HTTPPort), nil))
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
