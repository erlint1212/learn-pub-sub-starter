package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"log"
    amqp "github.com/rabbitmq/amqp091-go"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
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

	channel, err := connection.Channel()
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

	fmt.Println("Peril server connection succesful")

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	<-sigs

	fmt.Println("Peril is shutting down and close the server connection")
}
