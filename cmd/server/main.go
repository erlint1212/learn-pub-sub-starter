package main

import (
	"fmt"
	"log"
    amqp "github.com/rabbitmq/amqp091-go"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
)

func err_hand(err error) {
	log.Fatal(err)
}

func main() {
	fmt.Println("Starting Peril server...")

	connection_string := "amqp://guest:guest@localhost:5672/"

	connection, err := amqp.Dial(connection_string)
	if err != nil {
		err_hand(err)
	}

	defer func() {
		err := connection.Close()
		if err != nil {
			err_hand(fmt.Errorf("Error while closing server connection: %v", err))
		}
	}()

	channel, _, err := pubsub.DeclareAndBind(
		connection, 
		routing.ExchangePerilTopic,
		"game_logs", 
		"game_logs.*", 
		routing.Durable,
	)
	if err != nil {
		err_hand(err)
	}

	fmt.Println("Peril server connection succesful")

	gamelogic.PrintServerHelp()

	outerLoop:
		for {
			input := gamelogic.GetInput()
			if len(input) == 0 {
				continue
			}
			switch input[0] {
			case "pause":
				log.Println("Sending pause message")
				err = pubsub.PublishJSON(
					channel, 
					routing.ExchangePerilDirect, 
					routing.PauseKey, 
					routing.PlayingState{IsPaused: true},
				)
				if err != nil {
					err_hand(err)
				}
			case "resume":
				log.Println("Sending resume message")
				err = pubsub.PublishJSON(
					channel, 
					routing.ExchangePerilDirect, 
					routing.PauseKey, 
					routing.PlayingState{IsPaused: false},
				)
				if err != nil {
					err_hand(err)
				}
			case "quit":
				log.Println("Shutting down")
				break outerLoop
			default:
				log.Printf("Command \"%v\" not recognized", input[0])

			}
		}
	log.Println("Peril is shutting down and close the server connection")
}
