package main

import (
	"fmt"
	"time"
	"log"
    amqp "github.com/rabbitmq/amqp091-go"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"strconv"
)

func err_hand(err error) {
	log.Fatal(err)
}

func publishGameLog(gl routing.GameLog, channel *amqp.Channel) routing.AckType {
	err := pubsub.PublishGob(
		channel, 
		routing.ExchangePerilTopic, 
		routing.GameLogSlug + "." + gl.Username, 
		gl,
	)
	if err != nil {
		log.Println("ERROR: Error during publishing GameLog: %v", err)
		return routing.NackRequeue
	}

	return routing.Ack
}

func handlerPause(gs *gamelogic.GameState) func(ps routing.PlayingState) routing.AckType {
	return func(ps routing.PlayingState) routing.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return routing.Ack
	}
}

func handlerHandleMove(gs *gamelogic.GameState, publishCh *amqp.Channel) func(move gamelogic.ArmyMove) routing.AckType {
	return func(move gamelogic.ArmyMove) routing.AckType {
		defer fmt.Print("> ")
		MoveOutcome := gs.HandleMove(move)
		log.Println(MoveOutcome)
		switch MoveOutcome {
		case gamelogic.MoveOutComeSafe:
			return routing.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				publishCh, 
				routing.ExchangePerilTopic, 
				routing.WarRecognitionsPrefix + "." + gs.GetUsername(),
				gamelogic.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				fmt.Println(err)
				return routing.NackRequeue
			}
			return routing.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return routing.NackDiscard
		default:
			return routing.NackDiscard
	}
	}
}

func handlerConsumeAllWarMessages(gs *gamelogic.GameState, channel *amqp.Channel) func(rw gamelogic.RecognitionOfWar) routing.AckType {
	return func(rw gamelogic.RecognitionOfWar) routing.AckType {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(rw)
		var msg string
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return routing.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return routing.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			msg = fmt.Sprintf("%s won a war against %s", winner, loser)
			log.Println(msg)
			break
			// return routing.Ack
		case gamelogic.WarOutcomeYouWon:
			msg := fmt.Sprintf("%s won a war against %s", winner, loser)
			log.Println(msg)
			break
			// return routing.Ack
		case gamelogic.WarOutcomeDraw:
			msg = fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
			log.Println(msg)
			break
			// return routing.Ack
		default:
			log.Println("ERROR: Couldn't recognize the war outcome")
			return routing.NackDiscard
		}

		gamelog := routing.GameLog{
			time.Now(),
			msg,
			gs.GetUsername(),
		}
		result := publishGameLog(gamelog, channel)
		
		//log.Println("ERROR: Publishing gamelog did not produce a return")
		return result 
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

	err = pubsub.Subscribe(
		connection,
		routing.ExchangePerilDirect,
		routing.PauseKey + "." + gamestate.GetUsername(),
		routing.PauseKey,
		routing.Transient,
		handlerPause(gamestate),
		pubsub.UnmarshalJSON,
	)
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}

	err = pubsub.Subscribe(
		connection,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix + "." + gamestate.GetUsername(),
		routing.ArmyMovesPrefix + ".*",
		routing.Transient,
		handlerHandleMove(gamestate, publishCh),
		pubsub.UnmarshalJSON,
	)
	if err != nil {
		log.Fatalf("could not subscribe to army moves: %v", err)
	}

	err = pubsub.Subscribe(
		connection,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix + "." + gamestate.GetUsername(),
		routing.Durable,
		handlerConsumeAllWarMessages(gamestate, publishCh),
		pubsub.UnmarshalJSON,
	)
	if err != nil {
		err_hand(err)
	}

	outerLoop:
		for {
			input := gamelogic.GetInput()
			if len(input) == 0 {
				continue
			}
			defer fmt.Println("> ")
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
				if len(input) != 2 {
					log.Println("ERROR: Spam does not have a argument.")
					continue
				}
				spam_count, err := strconv.Atoi(input[1])
				if err != nil {
					log.Printf("ERROR: Error during convertion from string to int: %v\n", err)
					continue
				}
				successfully_sent := 0
				for i := 0; i < spam_count; i++ {
					msg := gamelogic.GetMaliciousLog()
					game_log := routing.GameLog{
						time.Now(),
						msg,
						gamestate.GetUsername(),
					}
					err = pubsub.PublishGob(
						publishCh, 
						routing.ExchangePerilTopic, 
						routing.GameLogSlug + "." + gamestate.GetUsername(), 
						game_log,
					)
					if err != nil {
						log.Printf("ERROR: Error during publishing of spam number %v\n", i)
						continue
					}
					successfully_sent += 1
				}
				log.Printf("Successfully sent %v spam messages, %v failed\n", successfully_sent, spam_count - successfully_sent)
			case "quit":
				log.Println("Shutting down")
				break outerLoop
			default:
				log.Printf("Command \"%v\" not recognized", input[0])

			}
		}


	log.Println("Peril is shutting down and close the server connection")
}

