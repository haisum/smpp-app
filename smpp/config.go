package smpp

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
)

type Conn struct {
	Id     string
	Url    string
	User   string
	Size   int
	Time   int
	Passwd string
	Pfxs   []string
}

type Config struct {
	AmqpUrl    string
	Conns      []Conn
	DefaultPfx string
}

func (c *Config) GetKeys() []string {
	keys := make([]string, 0)
	for _, con := range c.Conns {
		keys = append(keys, con.Pfxs...)
	}
	return keys
}

func (c *Config) GetConn(id string) (Conn, error) {
	var con Conn
	for _, con = range c.Conns {
		if con.Id == id {
			return con, nil
		}
	}

	return con, errors.New(fmt.Sprintf("Couldn't find key for connection %s.", id))
}

func (c *Config) LoadJSON(data []byte) error {
	err := json.Unmarshal(data, c)
	if err == nil {
		log.WithField("Config", c).Info("Loaded configuration")
	}
	return err
}

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
