package pubsub

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"encoding/gob"
	"bytes"
	"context"
)

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(val)
	if err != nil {
		return err
	}
	encoded_bytes := buffer.Bytes()

	msg := amqp.Publishing{
		ContentType: "application/gob",
		Body: encoded_bytes,
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, msg)
	if err != nil {
		return err
	}


	return nil
}

