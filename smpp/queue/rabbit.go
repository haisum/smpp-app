package queue

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"
)

var (
	q MQ
)

// MQ is interface implemented by messaging queue backend's client library
type MQ interface {
	Init(url string, ex string, pCount int) error
	Publish(key string, msg []byte, priority Priority) error
	Bind(group string, keys []string, handler Handler) error
	Close() error
}

// SetQueue sets queue equal to object that implements MQ interface. This function shouldn't be used unless you're testing.
// GetQueue takes care of setting a rabbitmq object if q is not set yet.
func setQueue(mq MQ) {
	q = mq
}

// GetQueue returns a rabbit object. It makes one connection per process life and reuses same rabbitmq connection.
func GetQueue(url string, ex string, pCount int) (MQ, error) {
	if q == nil {
		setQueue(&rabbit{})
		err := q.Init(url, ex, pCount)
		return q, err
	}
	return q, nil
}

// Priority represents priority of a message. O is default priority
// Higher number means higher priority. 10 is max priority after that, every number is considered to be 10
type Priority uint8

// Handler is a function which accepts deliveries channel and a error channel to indicate when processing is done
type Handler func(<-chan amqp.Delivery, chan error)

// rabbit implements MQ interface and holds host and port to connect to for rabbit mq and other properties for internal use
type rabbit struct {
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
func (r *rabbit) Init(url string, ex string, pCount int) error {
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
func (r *rabbit) Close() error {
	if err := r.Conn.Close(); err != nil {
		log.WithField("err", err).Error("AMQP connection close error")
		return err
	}

	defer log.Info("AMQP shutdown OK")

	// wait for handle() to exit
	return <-r.done
}

// Connects and makes channel to given amqp url
func (r *rabbit) connect() error {
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
	if err != nil {
		log.WithFields(log.Fields{
			"url": r.url,
			"err": err,
		}).Error("Failed to create channel.")
	}
	r.Ch.Qos(r.pCount, 0, false)
	return err
}

// Declares and Starts exchange ex.This can be called multiple times and wont re-create exchange once created. This uses direct exchange. See https://www.rabbitmq.com/tutorials/tutorial-four-go.html for details.
func (r *rabbit) startExchange() error {
	err := r.Ch.ExchangeDeclare(
		r.ex,     // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		amqp.Table{"x-max-priority": 10}, // arguments
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
func (r *rabbit) Publish(key string, msg []byte, priority Priority) error {
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
			"msg": string(msg),
			"err": err,
		}).Error("Error in publishing message. Retrying.")
		r.Init(r.url, r.ex, r.pCount)
		err = r.Ch.Publish(
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
			log.WithError(err).Error("Failed connecting to rabbitmq. Aborting.")
			os.Exit(2)
		}
	}
	return err
}

// Bind binds to queue defined by routing keys on exchange supplied to Init method.
// This method must be called after Init, otherwise it would fail.
func (r *rabbit) Bind(group string, keys []string, handler Handler) error {
	for _, k := range keys {
		k = fmt.Sprintf("%s-%s", group, k)
		q, err := r.Ch.QueueDeclare(
			k,     // name
			false, // durable
			false, // delete when usused
			false, // exclusive
			false, // no-wait
			amqp.Table{"x-max-priority": 10}, // arguments
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
