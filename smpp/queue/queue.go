package queue

import (
	"github.com/streadway/amqp"
	"log"
)

type Priority uint8

// Handler is a function which accepts deliveries channel and a error channel to indicate when processing is done
type Handler func(<-chan amqp.Delivery, chan error)

// Holds host and port to connect to for rabbit mq and other properties for internal use
type Rabbit struct {
	url  string
	ex   string
	conn *amqp.Connection
	ch   *amqp.Channel
	msgs <-chan amqp.Delivery
	done chan error
}

func (r *Rabbit) Init(url string, ex string) error {
	r.url = url
	r.ex = ex
	r.done = make(chan error)
	err := r.connect()
	if err != nil {
		return err
	}
	err = r.startExchange()
	return err
}

//call this with defer after calling Init function
func (r *Rabbit) Close() error {
	if err := r.conn.Close(); err != nil {
		log.Printf("AMQP connection close error: %s", err)
		return err
	}

	defer log.Printf("AMQP shutdown OK")

	// wait for handle() to exit
	return <-r.done
}

// Connects and makes channel to given amqp url
func (r *Rabbit) connect() error {
	var err error
	r.conn, err = amqp.Dial(r.url)
	if err != nil {
		log.Printf("[ERROR]: Failed to connect to rabbit mq on url %s. Error: %s.", r.url, err)
		return err
	}
	log.Print("Connection Successful. Creating channel.")
	r.ch, err = r.conn.Channel()
	if err != nil {
		log.Printf("[ERROR]: Failed to create channel. Error: %s.", r.url, err)
	}
	return err
}

// Declares and Starts exchange ex.This can be called multiple times and wont re-create exchange once created. This uses direct exchange. See https://www.rabbitmq.com/tutorials/tutorial-four-go.html for details.
func (r *Rabbit) startExchange() error {
	err := r.ch.ExchangeDeclare(
		r.ex,     // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		log.Printf("Error in creating exchange named %s. Error: %s.", r.ex, err)
	}
	return err
}

// Takes exchange name, routing key and message as parameters and publishes message
func (r *Rabbit) Publish(key string, msg []byte, priority Priority) error {
	err := r.ch.Publish(
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
		log.Printf("Error in publishing message %s. Error: %s", msg, err)
	}
	return err
}

// Binds to queue defined by routing keys on exchange supplied to Init method.
// This method must be called after Init, otherwise it would fail.
func (r *Rabbit) Bind(qName string, keys []string, handler Handler) error {
	q, err := r.ch.QueueDeclare(
		qName, // name
		false, // durable
		false, // delete when usused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Printf("Failed to create a queue. Error: %s", err)
		return err
	}
	for _, k := range keys {
		log.Printf("Binding queue %s to exchange %s with routing key %s",
			q.Name, r.ex, k)
		err = r.ch.QueueBind(
			q.Name, // queue name
			k,      // routing key
			r.ex,   // exchange
			false,
			nil)
		if err != nil {
			log.Printf("Failed to bind queue %s to exchange %s with routing key %k. Error: %s", q.Name, r.ex, k, err)
			return err
		}
	}

	r.msgs, err = r.ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto ack
		false,  // exclusive
		false,  // no local
		false,  // no wait
		nil,    // args
	)
	if err != nil {
		log.Printf("Failed to register a consumer. Error: %s", err)
	}
	go handler(r.msgs, r.done)
	return err
}
