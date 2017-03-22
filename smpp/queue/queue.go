package queue


var (
	q MQ
)



// Handler is a function which accepts deliveries channel and a error channel to indicate when processing is done
type Handler func(QueueDelivery)

// MQ is interface implemented by messaging queue backend's client library
type MQ interface {
	Publish(key string, msg []byte, priority Priority) error
	Bind(keys []string, handler Handler) error
	Close() error
}

//QueueDelivery is interface for delivery channel of queue
type QueueDelivery interface {
	Ack(multiple bool) error
	Reject(requeue bool) error
	Nack(multiple, requeue bool) error
	Body() []byte
}

// GetQueue returns a rabbit object. It makes one connection per process life and reuses same rabbitmq connection.
func GetQueue() MQ {
	return q
}

// ConnectRabbitMQ connects to rabbitmq and sets q to rabbit instance.
func ConnectRabbitMQ(url string, ex string, pCount int) (MQ, error) {
	q := &rabbit{}
	err := q.init(url, ex, pCount)
	return q, err
}

// Priority represents priority of a message. O is default priority
// Higher number means higher priority. 10 is max priority after that, every number is considered to be 10
type Priority uint8

