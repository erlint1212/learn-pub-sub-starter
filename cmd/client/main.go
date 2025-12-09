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

func handlerPause(gs *gamelogic.GameState) func(ps routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

func main() {
	log.Println("Starting Peril server...")

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

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		err_hand(err)
	}

	channel, _, err := pubsub.DeclareAndBind(
		connection, 
		routing.ExchangePerilDirect,
		routing.PauseKey + "." + username, 
		routing.PauseKey, 
		routing.Transient,
	)
	if err != nil {
		err_hand(err)
	}

	err = pubsub.PublishJSON(
		channel, 
		routing.ExchangePerilDirect, 
		routing.PauseKey, 
		routing.PlayingState{IsPaused: true},
	)
	if err != nil {
		err_hand(err)
	}

	gamestate := gamelogic.NewGameState(username)

	log.Println("Peril server connection succesful")

	pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilDirect,
		"pause." + username,
		routing.PauseKey,
		routing.Transient,
		handlerPause(gamestate),
	)

	outerLoop:
		for {
			input := gamelogic.GetInput()
			if len(input) == 0 {
				continue
			}
			switch input[0] {
			case "spawn":
				err := gamestate.CommandSpawn(input)
				if err != nil {
					err_hand(err)
				}
			case "move":
				_, err := gamestate.CommandMove(input)
				if err != nil {
					err_hand(err)
				}
				log.Printf("Moved unit \"%v\" to location \"%v\"", input[2], input[1])
			case "status":
				gamestate.CommandStatus()
			case "help":
				gamelogic.PrintClientHelp()
			case "spam":
				log.Println("Spamming not allowed yet!")
			case "quit":
				log.Println("Shutting down")
				break outerLoop
			default:
				log.Printf("Command \"%v\" not recognized", input[0])

			}
		}


	log.Println("Peril is shutting down and close the server connection")
}

