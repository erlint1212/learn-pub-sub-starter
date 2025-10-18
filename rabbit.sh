#!/usr/bin/env bash

start_or_run () {
    docker inspect rabbitmq_stomp > /dev/null 2>&1

    if [ $? -eq 0 ]; then
        #echo "Starting Peril RabbitMQ container..."
        echo "Starting STOMP RabbitMQ container..."
        #docker start peril_rabbitmq
        docker start rabbitmq_stomp
    else
        echo "Peril RabbitMQ container not found, creating a new one..."
        #docker run -d --name peril_rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3.13-management
        docker run -d --name rabbitmq_stomp -p 5672:5672 -p 15672:15672 -p 61613:61613 rabbitmq-stomp:latest
    fi
}

case "$1" in
    start)
        start_or_run
        ;;
    stop)
        echo "Stopping Peril RabbitMQ container..."
        docker stop rabbitmq_stomp
        ;;
    logs)
        echo "Fetching logs for Peril RabbitMQ container..."
        docker logs -f rabbitmq_stomp
        ;;
    *)
        echo "Usage: $0 {start|stop|logs}"
        exit 1
esac
