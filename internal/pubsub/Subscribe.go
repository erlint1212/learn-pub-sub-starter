package pubsub

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"encoding/gob"
	"bytes"
	//"context"
	"encoding/json"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"fmt"
	"log"
)

func UnmarshalGob[T any](data []byte) (T, error) {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	var gl T
	err := decoder.Decode(&gl)
	if err != nil {
		return gl, err
	}

	return gl, nil
}

func UnmarshalJSON[T any](data []byte) (T, error) {
	var target T
	err := json.Unmarshal(data, &target)
	if err != nil {
		return target, err
	}
	return target, nil
}

func Subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType routing.SimpleQueueType,
	handler func(T) routing.AckType,
	unmarshaller func([]byte) (T, error),
) error {
	ch, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("could not declare and bind queue: %v", err)
	}

	err = ch.Qos(10, 0, false)
	if err != nil {
		return fmt.Errorf("could not set qos (message count): %v", err)
	}
	msgs, err := ch.Consume(
		queue.Name, // queue
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("could not consume messages: %v", err)
	}

	go func() {
		defer ch.Close()
		for msg := range msgs {
			target, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				continue
			}
			acktype := handler(target)
			switch acktype {
			case routing.Ack:
				msg.Ack(false)
				log.Println("Acknowledge: Processed successfully.")
			case routing.NackRequeue:
				msg.Nack(false, true)
				log.Println("Nack and requeue: Not processed successfully, but should be requeued on the same queue to be processed again (retry).")
			case routing.NackDiscard:
				msg.Nack(false, false)
				log.Println("Nack and discard: Not processed successfully, and should be discarded (to a dead-letter queue if configured or just deleted entirely).")
			default:
				log.Println("Could not recognize Ack Type")
			}
		}
	}()
	return nil
}
