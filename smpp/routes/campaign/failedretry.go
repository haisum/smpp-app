package campaign

import (
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/message"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"bitbucket.org/codefreak/hsmpp/smpp/routes"
	"github.com/pkg/errors"
)

type retryRequest struct {
	CampaignID int64
	URL        string
	Token      string
}

type retryResponse struct {
	Count int
}

// Error represents a handler error. It provides methods for a HTTP status
// code and embeds the built-in error interface.
type Error interface {
	error
	Status() int
}

// StatusError represents an error with an associated HTTP status code.
type StatusError struct {
	Code int
	Err  error
}

// Allows StatusError to satisfy the error interface.
func (se StatusError) Error() string {
	return se.Err.Error()
}

// Returns our HTTP status code.
func (se StatusError) Status() int {
	return se.Code
}

type Handler struct {
	H   RouteHandler
	Env *Env
}

type RouteHandler interface {
	RequestStruct() interface{}
	Serve(e *Env, reqObj interface{}) (routes.ClientResponse, error)
}

type Retry struct {
	request  *retryRequest
	response *retryResponse
}

func (r *Retry) RequestStruct() interface{} {
	return &r.request
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.H(h.Env, w, r)
	if err != nil {
		switch e := err.(type) {
		case Error:
			// We can retrieve the status here and write out a specific
			// HTTP status code.
			log.Printf("HTTP %d - %s", e.Status(), e)
			http.Error(w, e.Error(), e.Status())
		default:
			// Any error types we don't specifically look out for default
			// to serving a HTTP 500
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		}
	} else {

	}
}

//RetryHandler accepts post request to restart all messages with status error of a campaign
func (r *Retry) Serve(e *Env, request routes.Request) (routes.Response, error) {
	uResp := retryResponse{}
	uReq, ok := request.(*retryRequest)
	if !ok {
		return nil, errors.New("invalid request")
	}
	var (
		u  user.User
		ok bool
	)
	// do in auth handler
	//if u, ok = routes.Authenticate(w, *r, uReq, uReq.Token, permission.RetryCampaign); !ok {
	//	return
	//}
	msgs, err := message.ListWithError(uReq.CampaignID)
	if err != nil {
		errors.Wrap(err, "error getting error messages")
		resp := routes.ClientResponse{}
		resp.Errors = []routes.ResponseError{
			{
				Type:    routes.ErrorTypeDB,
				Message: "Couldn't restart campaign.",
			},
		}
	}
	q := queue.Get()
	config, _ := smpp.GetConfig()
	keys := config.GetKeys(u.ConnectionGroup)
	var noKey string
	var group smpp.ConnGroup
	if group, err = config.GetGroup(u.ConnectionGroup); err != nil {
		log.WithField("ConnectionGroup", u.ConnectionGroup).Error("User's connection group doesn't exist in configuration.")
		resp := routes.ClientResponse{
			Errors: []routes.ResponseError{
				{
					Type:    routes.ErrorTypeConfig,
					Message: "User's connection group doesn't exist in configuration.",
				},
			},
			Request: uReq,
		}
		resp.Send(w, *r, http.StatusInternalServerError)
		return
	}
	errCh := make(chan error, 1)
	okCh := make(chan bool, len(msgs))
	burstCh := make(chan int, 1000)
	for _, msg := range msgs {
		go func(m message.Message) {
			m.QueuedAt = time.Now().UTC().Unix()
			m.Status = message.Queued
			errU := m.Update()
			if errU != nil {
				errCh <- errU
				return
			}
			noKey = group.DefaultPfx
			key := matchKey(keys, m.Dst, noKey)
			qItem := queue.Item{
				MsgID: m.ID,
				Total: m.Total,
			}
			respJSON, _ := qItem.ToJSON()
			errP := q.Publish(fmt.Sprintf("%s-%s", u.ConnectionGroup, key), respJSON, queue.Priority(m.Priority))
			if errP != nil {
				errCh <- errP
				return
			}
			okCh <- true
			//free one burst
			<-burstCh
		}(msg)
		//proceed if you can feed the burst channel
		burstCh <- 1
	}
	for i := 0; i < len(msgs); i++ {
		select {
		case <-errCh:
			log.WithFields(log.Fields{
				"error": err,
				"uReq":  uReq,
			}).Error("Couldn't publish message.")
			resp := routes.ClientResponse{
				Errors: []routes.ResponseError{
					{
						Type:    routes.ErrorTypeQueue,
						Message: "Couldn't queue message.",
					},
				},
				Request: uReq,
			}
			resp.Send(w, *r, http.StatusInternalServerError)
			return
		case <-okCh:
		}
	}
	log.Infof("%d campaign messages queued", len(msgs))
	uResp.Count = len(msgs)
	resp := routes.ClientResponse{
		Obj:     uResp,
		Ok:      true,
		Request: uReq,
	}
	resp.Send(w, *r, http.StatusOK)
}
