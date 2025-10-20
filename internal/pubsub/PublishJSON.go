package pubsub

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"encoding/json"
	"context"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	val_byte, err := json.Marshal(val)	
	if err != nil {
		return err
	}

	msg := amqp.Publishing{
		ContentType: "application/json",
		Body: val_byte,
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, msg)
	if err != nil {
		return err
	}


	return nil
}
