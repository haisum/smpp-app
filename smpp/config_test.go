package smpp

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	settings := []byte(`
{
    "AmqpUrl": "amqp://guest:guest@localhost:5672/",
    "HttpsPort": 8443,
    "Conns": [
        {
            "Id": "du-1",
            "Url": "192.168.0.105:2775",
            "User": "smppclient1",
            "Passwd": "password",
            "Pfxs": [
                "+97105","+97106", "+97107", "+97108",

                "+97106"
            ],
            "Size": 5,
            "Time": 1,
			"Fields" : {
				"ServiceType" : "sample service",
				"SourceAddrTON" : 2
			}
        },
        {
            "Id": "du-2",
            "Url": "192.168.0.105:2775",
            "User": "smppclient2",
            "Passwd": "password",
            "Pfxs": [
                "+97107",
                "+97108"
            ],
            "Size": 5,
            "Time": 1
        }
    ],
    "DefaultPfx": "+97105"
}
	`)
	err := ioutil.WriteFile("settings_test.json", settings, 0644)
	if err != nil {
		log.Fatalf("Couldn't create settings file in current directory. Error %s.", err)
	}
	status := m.Run()
	os.Remove("settings_test.json")
	os.Exit(status)
}

func TestConfig_GetKeys(t *testing.T) {
	var c Config
	err := c.LoadFile("settings_test.json")
	if err != nil {
		t.Fatalf("Couldn't load settings")
	}
	keys := c.GetKeys()
	expected := []string{"+97105", "+97106", "+97107", "+97108", "+97106", "+97107", "+97108"}
	if len(keys) != len(expected) {
		t.Fatalf("Count of returned keys (%d) not same as expected(%d).", len(keys), len(expected))
	}
	for x := range keys {
		if keys[x] != expected[x] {
			t.Fatalf("Expected key: %s. Given: %s", keys[x], expected[x])
		}
	}
}
func TestConfig_GetConn(t *testing.T) {
	var c Config
	err := c.LoadFile("settings_test.json")
	if err != nil {
		t.Fatalf("Couldn't load settings")
	}
	conn, _ := c.GetConn("du-1")
	if conn.Id != "du-1" || conn.Passwd != "password" || conn.User != "smppclient1" || conn.Pfxs[0] != "+97105" {
		t.Fatalf("Couldn't fetch conn for du-1. %+v", conn)
	}
	conn, _ = c.GetConn("du-2")
	if conn.Id != "du-2" || conn.Passwd != "password" || conn.User != "smppclient2" || conn.Pfxs[0] != "+97107" {
		t.Fatalf("Couldn't fetch conn for du-2. %+v", conn)
	}
	_, err = c.GetConn("somethingrandom")
	if err == nil {
		t.Fatalf("Didn't get error for random name")
	}
}
func TestConfig_LoadJSON(t *testing.T) {
	var c Config
	err := c.LoadFile("settings_test.json")
	if err != nil {
		t.Fatalf("Couldn't load settings")
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Couldn't marshall json. %s", err)
	}
	var newc Config
	newc.LoadJSON(data)
	if newc.DefaultPfx != c.DefaultPfx || newc.AmqpUrl != c.AmqpUrl || newc.Conns[0].Fields.SourceAddrTON != 2 {
		t.Fatalf("Loaded config doesn't match")
	}
}
func TestConfig_LoadFile(t *testing.T) {
	var c Config
	err := c.LoadFile("settings_test.json")
	if err != nil {
		t.Fatalf("Couldn't load settings")
	}
}
