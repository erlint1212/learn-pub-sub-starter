
package pubsub

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"fmt"
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType routing.SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	transient := false
	durable := false
	switch queueType {
	case routing.Transient:
		transient = true
	case routing.Durable:
		durable = true
	default:
		return nil, amqp.Queue{}, fmt.Errorf("queueType not recognized: %v", queueType)
	}

	args := amqp.Table{"x-dead-letter-exchange": "peril_dlx"}

	queue, err := channel.QueueDeclare(queueName, durable, transient, transient, false, args)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	fmt.Printf("Log: Name of queue \"%v\"\n", queue.Name)

	err = channel.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return channel, queue, err
}
