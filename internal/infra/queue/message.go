package queue

type Message interface {
	Body() []byte
	Ack() error
	Nack(multiple, requeue bool) error
}
