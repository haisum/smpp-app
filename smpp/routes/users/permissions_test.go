package users

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPermissions(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost/api/permissions", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	Permissions(w, r)
	if w.Code != 200 {
		t.Fatalf("Expected 200, returned code: %d", w.Code)
	}
	cType := w.Header().Get("Content-Type")
	if len(cType) < 1 || string(cType) != "application/json" {
		t.Fatalf("Invalid content type. %d %v", len(cType), cType)
	}
	expected := `{"Response":{"Permissions":["AddUser","SuspendUser"]},"Errors":null,"Ok":true,"Request":{"Url":"/api/permissions"}}`
	if expected != w.Body.String() {
		t.Fatalf("Expected: %s, Got: %s", expected, w.Body.String())
	}
	// XML
	w = httptest.NewRecorder()
	r.Header["Content-Type"] = []string{"text/xml"}
	Permissions(w, r)
	if w.Code != 200 {
		t.Fatalf("Expected 200, returned code: %d", w.Code)
	}
	cType = w.Header().Get("Content-Type")
	if len(cType) < 1 || string(cType) != "text/xml;charset=UTF-8" {
		t.Fatalf("Invalid content type. %v", cType)
	}
	expected = `<Response><Obj><Permissions>AddUser</Permissions><Permissions>SuspendUser</Permissions></Obj><Ok>true</Ok><Request><Url>/api/permissions</Url></Request></Response>`
	if expected != w.Body.String() {
		t.Fatalf("Expected: %s, Got: %s", expected, w.Body.String())
	}
}
