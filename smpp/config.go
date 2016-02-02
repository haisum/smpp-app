package smpp

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
)

// Conn represents configuration specific to a single smpp connection
type Conn struct {
	Id     string
	Url    string
	User   string
	Size   int32
	Time   int
	Passwd string
	Pfxs   []string
}

// Config represents all settings defined in settings file
type Config struct {
	AmqpUrl    string
	Conns      []Conn
	DefaultPfx string
	HttpsPort  int
}

// Returns all prefixes defined by all the connections
func (c *Config) GetKeys() []string {
	keys := make([]string, 0)
	for _, con := range c.Conns {
		keys = append(keys, con.Pfxs...)
	}
	return keys
}

// Returns a connection with given id
func (c *Config) GetConn(id string) (Conn, error) {
	var con Conn
	for _, con = range c.Conns {
		if con.Id == id {
			return con, nil
		}
	}

	return con, fmt.Errorf("Couldn't find key for connection %s.", id)
}

// Loads config from json byte stream
func (c *Config) LoadJSON(data []byte) error {
	err := json.Unmarshal(data, c)
	return err
}

// Loads config from given file
func (c *Config) LoadFile(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.WithField("filename", filename).Error("Could not read from file.")
		return err
	}
	err = c.LoadJSON(b)
	if err != nil {
		log.WithField("err", err).Error("Couldn't load json from settings file.")
		con := Conn{}
		con.Pfxs = []string{"+97105", "+97106"}
		c.Conns = []Conn{con}
		c.DefaultPfx = "+97105"
		d, err := json.MarshalIndent(c, "", "    ")
		if err == nil {
			log.Info("Expected format:\n%s", d)
		}
	}
	return err
}
