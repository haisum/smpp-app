package models

import "testing"

func TestGetToken(t *testing.T) {
	tok1, err := CreateToken("user1", 0)
	if err != nil {
		t.Fatal(err)
	}
	tok2, err := CreateToken("user2", 0)
	if err != nil {
		t.Fatal(err)
	}
	gTok, err := GetToken(tok1)
	if err != nil {
		t.Fatal(err)
	}
	if gTok.Username != "user1" {
		t.Fatalf("Expected username user1. Got %s", gTok.Username)
	}

	gTok, err = GetToken(tok2)
	if err != nil {
		t.Fatal(err)
	}
	if gTok.Username != "user2" {
		t.Fatalf("Expected username user2. Got %s", gTok.Username)
	}
}
func TestToken_Delete(t *testing.T) {
	tok, err := CreateToken("user1", 0)
	if err != nil {
		t.Fatal(err)
	}
	gTok, err := GetToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	err = gTok.Delete()
	if err != nil {
		t.Fatal(err)
	}
	_, err = GetToken(tok)
	if err == nil {
		t.Fatal("Token was not deleted.")
	}
}
func TestToken_DeleteAll(t *testing.T) {
	tok1, err := CreateToken("user1", 0)
	if err != nil {
		t.Fatal(err)
	}
	tok2, err := CreateToken("user1", 0)
	if err != nil {
		t.Fatal(err)
	}
	gTok, err := GetToken(tok1)
	if err != nil {
		t.Fatal(err)
	}
	err = gTok.DeleteAll()
	if err != nil {
		t.Fatal(err)
	}
	_, err = GetToken(tok2)
	if err == nil {
		t.Fatal("Expected to get error, still got tok2")
	}
}
