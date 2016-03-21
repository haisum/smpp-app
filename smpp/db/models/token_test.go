package models

import (
	"bitbucket.com/codefreak/hsmpp/smpp/db"
	"testing"
)

func TestGetToken(t *testing.T) {
	s, err := db.GetSession()
	if err != nil {
		t.Fatal(err)
	}
	tok1, err := CreateToken(s, "user1")
	if err != nil {
		t.Fatal(err)
	}
	tok2, err := CreateToken(s, "user2")
	if err != nil {
		t.Fatal(err)
	}
	gTok, err := GetToken(s, tok1)
	if err != nil {
		t.Fatal(err)
	}
	if gTok.Username != "user1" {
		t.Fatalf("Expected username user1. Got %s", gTok.Username)
	}

	gTok, err = GetToken(s, tok2)
	if err != nil {
		t.Fatal(err)
	}
	if gTok.Username != "user2" {
		t.Fatalf("Expected username user2. Got %s", gTok.Username)
	}
}
func TestToken_Delete(t *testing.T) {
	s, err := db.GetSession()
	if err != nil {
		t.Fatal(err)
	}
	tok, err := CreateToken(s, "user1")
	if err != nil {
		t.Fatal(err)
	}
	gTok, err := GetToken(s, tok)
	if err != nil {
		t.Fatal(err)
	}
	err = gTok.Delete(s)
	if err != nil {
		t.Fatal(err)
	}
	_, err = GetToken(s, tok)
	if err == nil {
		t.Fatal("Token was not deleted.")
	}
}
func TestToken_DeleteAll(t *testing.T) {
	s, err := db.GetSession()
	if err != nil {
		t.Fatal(err)
	}
	tok1, err := CreateToken(s, "user1")
	if err != nil {
		t.Fatal(err)
	}
	tok2, err := CreateToken(s, "user1")
	if err != nil {
		t.Fatal(err)
	}
	gTok, err := GetToken(s, tok1)
	if err != nil {
		t.Fatal(err)
	}
	err = gTok.DeleteAll(s)
	if err != nil {
		t.Fatal(err)
	}
	_, err = GetToken(s, tok2)
	if err == nil {
		t.Fatal("Expected to get error, still got tok2")
	}
}
