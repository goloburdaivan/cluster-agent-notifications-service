package queue

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
)

type RabbitMessage struct {
	body     []byte
	ackFunc  func() error
	nackFunc func(multiple, requeue bool) error
}

func (m *RabbitMessage) Body() []byte {
	return m.body
}

func (m *RabbitMessage) Ack() error {
	return m.ackFunc()
}

func (m *RabbitMessage) Nack(multiple, requeue bool) error {
	return m.nackFunc(multiple, requeue)
}

type rabbitQueue struct {
	name string
	conn *amqp.Connection
	ch   *amqp.Channel
}

func (r *rabbitQueue) Consume(ctx context.Context) (<-chan Message, error) {
	messages, err := r.ch.Consume(
		r.name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return nil, err
	}

	out := make(chan Message)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-messages:
				if !ok {
					return
				}

				out <- &RabbitMessage{
					body: msg.Body,
					ackFunc: func() error {
						return msg.Ack(false)
					},
					nackFunc: func(multiple, requeue bool) error {
						return msg.Nack(multiple, requeue)
					},
				}
			}
		}
	}()

	return out, nil
}

func (r *rabbitQueue) Close() error {
	return r.ch.Close()
}

func NewRabbitQueue(
	name string,
	conn *amqp.Connection,
) Queue {
	queue := &rabbitQueue{
		name: name,
		conn: conn,
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	err = ch.Qos(20, 0, false)
	if err != nil {
		log.Fatal(err)
	}

	_, err = ch.QueueDeclare(
		name,
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		log.Fatal(err)
	}

	queue.ch = ch

	return queue
}
