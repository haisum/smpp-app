package models

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/fresh"
	"log"
	"reflect"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	s, err := db.GetSession()
	if err != nil {
		log.Fatalf("%s", err)
	}
	db.DBName = db.DBTestName
	fresh.Drop(s, db.DBName)
	err = fresh.Create(s, db.DBName)
	if err != nil {
		log.Fatalf("%s", err)
	}
	m.Run()
}

func TestGetUser(t *testing.T) {
	t.Skip("Skipping because it's covered in User_Add")
}
func TestGetUsers(t *testing.T) {
	s, err := db.GetSession()
	if err != nil {
		t.Fatalf("Failed getting session. %s", err)
	}
	users := []User{
		User{
			Username:        "ahaisum1",
			Password:        "password123",
			Name:            "Haisum Bhatti",
			Email:           "haisumbhatti@gmail.com",
			NightStartAt:    "07:00 AM",
			NightEndAt:      "07:00 PM",
			ConnectionGroup: "myconn",
			Permissions:     []Permission{PermSuspendUser},
			RegisteredAt:    time.Now().Unix(),
		},
		User{
			Username:        "bhaisum2",
			Password:        "password123",
			Name:            "Haisum Bhatti",
			Email:           "haisumbhatti@gmail.com",
			NightStartAt:    "07:00 AM",
			NightEndAt:      "07:00 PM",
			ConnectionGroup: "myconn",
			Permissions:     []Permission{PermAddUser, PermSuspendUser},
			RegisteredAt:    time.Now().Unix(),
		},
	}
	var lastRegTime int64
	for _, user := range users {
		time.Sleep(time.Second * 1)
		user.RegisteredAt = time.Now().Unix()
		_, err := user.Add(s)
		if err != nil {
			t.Fatalf("%s", err)
		}
		lastRegTime = user.RegisteredAt
	}
	users, err = GetUsers(s, UserCriteria{})
	if len(users) < 2 {
		t.Fatal("Not enough users returned.")
	}
	if users[0].Username != "bhaisum2" {
		t.Fatalf("Latest record not returned as first record. %+v", users)
	}
	if users[1].Username != "ahaisum1" {
		t.Fatalf("Order isn't correct. %+v", users)
	}
	users, err = GetUsers(s, UserCriteria{OrderByDir: "ASC"})
	if len(users) < 2 {
		t.Fatal("Not enough users returned.")
	}
	if users[0].Username != "ahaisum1" {
		t.Fatalf("Old record not returned as first record. %+v", users)
	}
	if users[1].Username != "bhaisum2" {
		t.Fatalf("Order isn't correct. %+v", users)
	}
	users, err = GetUsers(s, UserCriteria{OrderByKey: "Username", OrderByDir: "ASC"})
	if len(users) < 2 {
		t.Fatal("Not enough users returned.")
	}
	if users[0].Username != "ahaisum1" {
		t.Fatalf("Old record not returned as first record. %+v", users)
	}
	if users[1].Username != "bhaisum2" {
		t.Fatalf("Order isn't correct. %+v", users)
	}
	users, err = GetUsers(s, UserCriteria{OrderByKey: "Username"})
	if len(users) < 2 {
		t.Fatal("Not enough users returned.")
	}
	if users[0].Username != "bhaisum2" {
		t.Fatalf("Old record not returned as first record. %+v", users)
	}
	if users[1].Username != "ahaisum1" {
		t.Fatalf("Order isn't correct. %+v", users)
	}
	users, err = GetUsers(s, UserCriteria{Username: "ahaisum1"})
	if len(users) != 1 {
		t.Fatal("No or more users returned.")
	}
	if users[0].Username != "ahaisum1" {
		t.Fatal("Username filter didn't work")
	}
	users, err = GetUsers(s, UserCriteria{Permissions: []Permission{PermAddUser, PermSuspendUser}})
	if len(users) != 1 {
		t.Fatalf("Permission filter returned none or more than expected users. %+v", users)
	}
	if users[0].Username != "bhaisum2" {
		t.Fatalf("Permission filter didn't work. %+v", users[0])
	}
	users[0].Id = ""
	users[0].Username = "lateregister"
	users[0].RegisteredAt = lastRegTime + 1
	users[0].Add(s)
	users[0].Permissions = []Permission{PermAddUser}
	users, err = GetUsers(s, UserCriteria{RegisteredAfter: lastRegTime})
	if len(users) != 1 {
		t.Fatalf("RegisteredAfter filter returned none or more than expected users. %+v", users)
	}
	if users[0].Username != "lateregister" {
		t.Fatal("RegisteredAfter filter didn't work")
	}
	users, err = GetUsers(s, UserCriteria{RegisteredBefore: lastRegTime + 1, Permissions: []Permission{PermAddUser}})
	if len(users) != 1 {
		t.Fatalf("RegisteredBefore + Permission filter returned none or more than expected users. %+v", users)
	}
	if users[0].Username != "bhaisum2" {
		t.Fatal("RegisteredBefore + Permission filter didn't work. %+v", users[0])
	}
	users, err = GetUsers(s, UserCriteria{Username: "lateregister", RegisteredBefore: lastRegTime + 1, Permissions: []Permission{PermAddUser}})
	if len(users) != 0 {
		t.Fatalf("No results were expected yet received %v.", users)
	}
}
func TestUser_Add(t *testing.T) {
	s, err := db.GetSession()
	if err != nil {
		t.Fatalf("Failed getting session. %s", err)
	}
	u := User{
		Username:        "haisum1",
		Password:        "password123",
		Name:            "Haisum Bhatti",
		Email:           "haisumbhatti@gmail.com",
		NightStartAt:    "07:00 AM",
		NightEndAt:      "07:00 PM",
		ConnectionGroup: "myconn",
		Permissions:     []Permission{PermAddUser, PermSuspendUser},
		RegisteredAt:    time.Now().Unix(),
	}
	u.Id, err = u.Add(s)
	if err != nil {
		t.Fatalf("Error occured with valid data. Error: %s", err)
	}
	u2, err := GetIdUser(s, u.Id)
	if err != nil {
		t.Fatalf("Error occured in getting user. Error: %s", err)
	}
	if !reflect.DeepEqual(u, u2) {
		t.Fatalf("Added user and returned user aren't equal. Added: %+v Returned: %+v", u, u2)
	}
	// users for validation check
	users := []User{
		User{
			Username:        "haisum2",
			Password:        "123",
			Name:            "Haisum Bhatti",
			Email:           "haisumbhatti@gmail.com",
			NightStartAt:    "07:00 AM",
			NightEndAt:      "07:00 PM",
			ConnectionGroup: "myconn",
			Permissions:     []Permission{PermAddUser, PermSuspendUser},
			RegisteredAt:    time.Now().Unix(),
		},
		User{
			Username:        "haisum3",
			Password:        "123",
			Name:            "Haisum Bhatti",
			Email:           "haisumbhattigmail.com",
			NightStartAt:    "07:00 AM",
			NightEndAt:      "07:00 PM",
			ConnectionGroup: "myconn",
			Permissions:     []Permission{PermSuspendUser},
			RegisteredAt:    time.Now().Unix(),
		}, User{
			Username:        "haisum4",
			Password:        "password123",
			Name:            "Haisum Bhatti",
			Email:           "haisumbhatti@gmail.com",
			NightStartAt:    "7:00 M",
			NightEndAt:      "07:00 PM",
			ConnectionGroup: "myconn",
			Permissions:     []Permission{PermAddUser},
			RegisteredAt:    time.Now().Unix(),
		}, User{
			Username:        "haisum5",
			Password:        "123sadf",
			Name:            "Haisum Bhatti",
			Email:           "haisumbhatti@gmail.com",
			NightStartAt:    "07:00 PM",
			NightEndAt:      "7:00 M",
			ConnectionGroup: "myconn",
			Permissions:     []Permission{"perm1", "perm2"},
			RegisteredAt:    time.Now().Unix(),
		}, User{
			Username:        "haisum6",
			Password:        "123sadf",
			Name:            "Haisum Bhatti",
			Email:           "haisumbhatti@gmail.com",
			NightStartAt:    "07:00 PM",
			NightEndAt:      "07:00 AM",
			ConnectionGroup: "myconn",
			Permissions:     []Permission{"perm1", "perm2"},
			RegisteredAt:    time.Now().Unix(),
		}, User{
			Username:        "haisum7",
			Password:        "123sadf",
			Name:            "Haisum Bhatti",
			Email:           "haisumbhatti@gmail.com",
			NightStartAt:    "07:00 AM",
			NightEndAt:      "07:00 PM",
			ConnectionGroup: "myconn",
			Permissions:     []Permission{"perm1", "perm2"},
			RegisteredAt:    time.Now().Unix(),
		}, User{
			Username:        "haisum1",
			Password:        "123sadf",
			Name:            "Haisum Bhatti",
			Email:           "haisumbhatti@gmail.com",
			NightStartAt:    "07:00 AM",
			NightEndAt:      "07:00 PM",
			ConnectionGroup: "myconn",
			Permissions:     []Permission{PermAddUser},
			RegisteredAt:    time.Now().Unix(),
		},
	}
	for _, user := range users {
		user.Id, err = user.Add(s)
		if err == nil {
			t.Fatalf("User added with invalid data. User: %+v", user)
		}
	}
}
func TestUser_Update(t *testing.T) {
	s, err := db.GetSession()
	if err != nil {
		t.Fatalf("Failed getting session. %s", err)
	}
	u := User{
		Username:        "haisum8",
		Password:        "password123",
		Name:            "Haisum Bhatti",
		Email:           "haisumbhatti@gmail.com",
		NightStartAt:    "07:00 AM",
		NightEndAt:      "07:00 PM",
		ConnectionGroup: "myconn",
		Permissions:     []Permission{PermAddUser, PermSuspendUser},
		RegisteredAt:    time.Now().Unix(),
	}
	u.Id, err = u.Add(s)
	if err != nil {
		t.Fatalf("Error occured while adding user. Error: %s", err)
	}
	u.Name = "Haisum Mussawir"
	err = u.Update(s, false)
	if err != nil {
		t.Fatalf("Error occured while updating user. Error: %s", err)
	}
	u2, _ := GetUser(s, u.Username)
	if u2.Name != u.Name {
		t.Fatal("User record didn't update.")
	}
	u.Password = "newpasswordhere"
	err = u.Update(s, true)
	if err != nil {
		t.Fatal("Couldn't update password.")
	}
	if !u.Auth("newpasswordhere") {
		t.Fatal("Couldn't authenticate with new password.")
	}
	u.Email = "helloworldsa"
	err = u.Update(s, false)
	if err == nil {
		t.Fatal("Invalid user data got updated.")
	}
}
