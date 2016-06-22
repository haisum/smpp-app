package influx

import (
	"fmt"
	"sync"
	"time"

	influxdb "github.com/influxdata/influxdb/client/v2"
)

const (
	// DBNAME is influxdb database name
	DBNAME = "hsmppdb"
)

// Tags are influxdb tag names that we want to give to a metric
type Tags map[string]string

// Fields are values for metric
type Fields map[string]interface{}

// Point is measurement with fields and tags at a specific time
type Point struct {
	Measurement string
	Tags        Tags
	Fields      Fields
	Time        time.Time
}

// Client is interface to influxdb database
type Client interface {
	// AddPoint adds given point to a batch
	AddPoint(p *Point) error
	// AddPoints adds multiple points to batch
	AddPoints(ps []*Point) error
	// TotalPoints returns number of points in batch not written yet
	TotalPoints() int
	// Write flushes batch to influxdb and resets batch
	Write() error
	// Exec runs a query on influxdb
	Exec(cmd string) error
	// Closes frees influxdb client resource
	Close() error
}

type client struct {
	bp    influxdb.BatchPoints
	c     influxdb.Client
	total int
	sync.Mutex
}

var (
	c Client
)

//GetClient returns client object
func GetClient() (Client, error) {
	if c == nil {
		return c, fmt.Errorf("Client hasn't been connected yet. Please call Connect before getting client.")
	}
	return c, nil
}

// Connect connects to influxdb, creates new database if it doesn't already exist, it also creates a new batch
// It returns pointer to instance of client object as Client
func Connect(addr, username, password string) (Client, error) {
	var (
		err error
		cl  = &client{}
	)
	cl.c, err = influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     addr,
		Username: username,
		Password: password,
	})
	if err != nil {
		return cl, err
	}
	query := "CREATE DATABASE IF NOT EXISTS " + DBNAME + ";"
	err = cl.Exec(query)
	if err != nil {
		return cl, err
	}
	cl.bp, err = influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database: DBNAME,
	})
	c = cl
	return cl, err
}

// Exec runs a query on influxdb
func (cl *client) Exec(cmd string) error {
	q := influxdb.Query{
		Command:  cmd,
		Database: DBNAME,
	}
	_, err := cl.c.Query(q)
	return err
}

// AddPoint adds given point to a batch
func (cl *client) AddPoint(p *Point) error {
	np, err := influxdb.NewPoint(p.Measurement, map[string]string(p.Tags), map[string]interface{}(p.Fields), p.Time)
	if err != nil {
		return err
	}
	cl.Lock()
	cl.total = cl.total + 1
	cl.bp.AddPoint(np)
	cl.Unlock()
	return nil
}

// AddPoints adds multiple points to a batch
func (cl *client) AddPoints(ps []*Point) error {
	var nps []*influxdb.Point
	for _, p := range ps {
		np, err := influxdb.NewPoint(p.Measurement, map[string]string(p.Tags), map[string]interface{}(p.Fields), p.Time)
		if err != nil {
			return err
		}
		nps = append(nps, np)
	}
	cl.Lock()
	cl.total = cl.total + len(nps)
	cl.bp.AddPoints(nps)
	cl.Unlock()
	return nil
}

// TotalPoints returns number of unwritten points
func (cl *client) TotalPoints() int {
	return cl.total
}

// Write flushes batch to influxdb and resets batch
func (cl *client) Write() error {
	cl.Lock()
	err := cl.c.Write(cl.bp)
	if err != nil {
		return err
	}
	cl.bp, err = influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database: DBNAME,
	})
	cl.total = 0
	cl.Unlock()
	return err
}

// Closes frees influxdb client resource
func (cl *client) Close() error {
	return cl.c.Close()
}
