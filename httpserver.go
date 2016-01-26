package main

import (
	"encoding/json"
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/queue"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type SendResponse struct {
	Errors  []string
	Request smpp.QueueItem
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
	var q queue.Rabbit
	err := q.Init("amqp://guest:guest@localhost:5672/", "TestExchange")
	if err != nil {
		log.Fatalf("Error occured in connecting to rabbitmq. %s", err)
	}

	keys := []string{"firstroutingkey"}
	noKey := "firstroutingkey"

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

		resp.Request = smpp.QueueItem{
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
				log.Printf("Error in formatting json response %v. Error: %s", resp, err)
				http.Error(w, "Internal server error. See logs for details.", http.StatusInternalServerError)
				return
			}
			http.Error(w, string(respJson[:]), http.StatusBadRequest)
			return
		}
		rJson, err := resp.Request.ToJSON()
		if err != nil {
			log.Printf("Error in formatting json request %v. Error: %s", resp.Request, err)
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
			log.Printf("Error in unmarshalling response: %v. %s", resp, err)
			http.Error(w, "Internal server error occured. See http server logs for details.", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(b)
		if err != nil {
			log.Printf("Error in writing response. %s", err)
			http.Error(w, "Internal server error occured. See http server logs for details.", http.StatusInternalServerError)
		}
	})
	log.Printf("Listening on http://127.0.0.1:8080.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
