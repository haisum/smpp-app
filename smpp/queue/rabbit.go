package queue

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"
)

var (
	q *Rabbit
)

// GetQueue returns a Rabbit object. It makes one connection per process life and reuses same rabbitmq connection.
func GetQueue(url string, ex string, pCount int) (*Rabbit, error) {
	if q == nil {
		q = &Rabbit{}
		err := q.Init(url, ex, pCount)
		return q, err
	}
	return q, nil
}

// Priority represents priority of a message
// Higher number means higher priority. Four priorities are supported:
// priority.Low
// priority.Normal
// priority.Medium
// priority.High
// Passing other numbers doesn't halt program but may result in undefined behavior
type Priority uint8

// Handler is a function which accepts deliveries channel and a error channel to indicate when processing is done
type Handler func(<-chan amqp.Delivery, chan error)

// Rabbit holds host and port to connect to for rabbit mq and other properties for internal use
type Rabbit struct {
	url    string
	ex     string
	pCount int
	Conn   *amqp.Connection
	Ch     *amqp.Channel
	msgs   <-chan amqp.Delivery
	done   chan error
}

// Init takes url, exchange name and burst count as argument and
// creates a new exchange, on rabbitmq url
func (r *Rabbit) Init(url string, ex string, pCount int) error {
	r.url = url
	r.ex = ex
	r.pCount = pCount
	r.done = make(chan error)
	err := r.connect()
	if err != nil {
		return err
	}
	err = r.startExchange()
	return err
}

//Close closes the connection to rabbitmq
//call this with defer after calling Init function
func (r *Rabbit) Close() error {
	if err := r.Conn.Close(); err != nil {
		log.WithField("err", err).Error("AMQP connection close error")
		return err
	}

	defer log.Info("AMQP shutdown OK")

	// wait for handle() to exit
	return <-r.done
}

// Connects and makes channel to given amqp url
func (r *Rabbit) connect() error {
	var err error
	r.Conn, err = amqp.Dial(r.url)
	if err != nil {
		log.WithFields(log.Fields{
			"url": r.url,
			"err": err,
		}).Error("Failed to connect to rabbit mq.")
		return err
	}
	log.Info("Connection Successful. Creating channel.")
	r.Ch, err = r.Conn.Channel()
	r.Ch.Qos(r.pCount, 0, false)
	if err != nil {
		log.WithFields(log.Fields{
			"url": r.url,
			"err": err,
		}).Error("Failed to create channel.")
	}
	return err
}

// Declares and Starts exchange ex.This can be called multiple times and wont re-create exchange once created. This uses direct exchange. See https://www.rabbitmq.com/tutorials/tutorial-four-go.html for details.
func (r *Rabbit) startExchange() error {
	err := r.Ch.ExchangeDeclare(
		r.ex,     // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		log.WithFields(log.Fields{
			"ex":  r.ex,
			"err": err,
		}).Error("Error in creating exchange.")
	}
	return err
}

// Publish takes exchange name, routing key and message as parameters and publishes message
func (r *Rabbit) Publish(key string, msg []byte, priority Priority) error {
	err := r.Ch.Publish(
		r.ex,  // exchange
		key,   // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msg,
			Priority:    uint8(priority),
		})
	if err != nil {
		log.WithFields(log.Fields{
			"msg": msg,
			"err": err,
		}).Error("Error in publishing message.")
	}
	return err
}

// Bind binds to queue defined by routing keys on exchange supplied to Init method.
// This method must be called after Init, otherwise it would fail.
func (r *Rabbit) Bind(group string, keys []string, handler Handler) error {
	for _, k := range keys {
		k = fmt.Sprintf("%s-%s", group, k)
		q, err := r.Ch.QueueDeclare(
			k,     // name
			false, // durable
			false, // delete when usused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			log.WithField("err", err).Error("Failed to create a queue.")
			return err
		}
		log.WithFields(log.Fields{
			"q.Name": q.Name,
			"r.ex":   r.ex,
			"k":      k,
		}).Info("Binding queue.")
		err = r.Ch.QueueBind(
			q.Name, // queue name
			k,      // routing key
			r.ex,   // exchange
			false,
			nil)
		if err != nil {
			log.WithFields(log.Fields{
				"q.Name": q.Name,
				"r.ex":   r.ex,
				"k":      k,
				"err":    err,
			}).Error("Failed to bind queue.")
			return err
		}

		r.msgs, err = r.Ch.Consume(
			q.Name, // queue
			"",     // consumer
			false,  // auto ack
			false,  // exclusive
			false,  // no local
			false,  // no wait
			nil,    // args
		)
		if err != nil {
			log.WithField("err", err).Error("Failed to register a consumer.")
			return err
		}
		go handler(r.msgs, r.done)
	}
	return nil
}
