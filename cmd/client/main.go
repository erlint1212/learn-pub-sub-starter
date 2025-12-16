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

func handlerHandleMove(gs *gamelogic.GameState) func(move gamelogic.ArmyMove) {
	return func(move gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		MoveOutcome := gs.HandleMove(move)
		log.Println(MoveOutcome)
	}
}

func main() {
	log.Println("Starting Peril server...")

	const connection_string = "amqp://guest:guest@localhost:5672/"

	connection, err := amqp.Dial(connection_string)
	if err != nil {
		err_hand(err)
	}

	defer func() {
		err := connection.Close()
		if err != nil {
			err_hand(fmt.Errorf("Error while closing server connection: %v", err))
		}
		log.Println("Server connection successfully closed")
	}()

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		err_hand(err)
	}

	_, _, err = pubsub.DeclareAndBind(
		connection, 
		routing.ExchangePerilDirect,
		routing.PauseKey + "." + username, 
		routing.PauseKey, 
		routing.Transient,
	)
	if err != nil {
		err_hand(err)
	}

	publishCh, err := connection.Channel()
	
	err = pubsub.PublishJSON(
		publishCh, 
		routing.ExchangePerilDirect, 
		routing.PauseKey, 
		routing.PlayingState{},
	)
	if err != nil {
		err_hand(err)
	}

	gamestate := gamelogic.NewGameState(username)

	log.Println("Peril server connection succesful")

	err = pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilDirect,
		routing.PauseKey + "." + gamestate.GetUsername(),
		routing.PauseKey,
		routing.Transient,
		handlerPause(gamestate),
	)
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}

	err = pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix + "." + gamestate.GetUsername(),
		routing.ArmyMovesPrefix + ".*",
		routing.Transient,
		handlerHandleMove(gamestate),
	)
	if err != nil {
		log.Fatalf("could not subscribe to army moves: %v", err)
	}

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
					fmt.Println(err)
					continue
				}
				log.Printf("Spawned unit \"%v\"\n", input[1])
			case "move":
				mv, err := gamestate.CommandMove(input)
				if err != nil {
					fmt.Println(err)
					continue
				}
				log.Printf("Moved unit \"%v\" to location \"%v\"\n", input[2], input[1])
				err = pubsub.PublishJSON(
					publishCh, 
					routing.ExchangePerilTopic, 
					routing.ArmyMovesPrefix + "." + mv.Player.Username, 
					mv,
				)
				if err != nil {
					fmt.Println(err)
					continue
				}
				log.Printf("Move was published successfully")

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

