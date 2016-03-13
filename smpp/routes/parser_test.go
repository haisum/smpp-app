package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testRequest struct {
	Action string
	Data   []string
	Url    string
}

type testResp struct {
	Result string
}

func TestParseRequest(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var tReq testRequest
		err := ParseRequest(*r, &tReq)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		tReq.Url = r.URL.RequestURI()
		tResp := testResp{
			Result: "pass",
		}
		response := Response{
			Obj:     tResp,
			Request: tReq,
			Ok:      true,
		}
		b, cType, err := MakeResponse(*r, response)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", cType)
		fmt.Fprint(w, string(b))
	}
	tReq := testRequest{
		Action: "Edit",
		Data:   []string{"var1", "var2"},
	}
	b, _ := json.Marshal(tReq)
	body := strings.NewReader(string(b[:]))
	req, err := http.NewRequest("GET", "http://localhost/api", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header["Content-Type"] = []string{"application/json"}
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != 200 {
		t.Fatalf("Error code expected 200. Got %d", w.Code)
	}
	resp := Response{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatal(err)
	}
	tReq.Url = "/api"
	data := resp.Request.(map[string]interface{})["Data"].([]interface{})
	if data[0].(string) != tReq.Data[0] || data[1].(string) != tReq.Data[1] {
		t.Fatalf("Response/Request data didn't match. Given: %v Received: %v", resp.Request.(map[string]interface{})["Data"], tReq.Data)
	}
}
