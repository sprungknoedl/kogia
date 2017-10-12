package main

import (
	"log"

	"github.com/streadway/amqp"
)

type MockInput struct{}

func (in MockInput) GetMetric(name string) (int, error) {
	return 123, nil
}

type AMQPInput struct {
	conn *amqp.Connection
}

func NewAMQPInput(addr string) AMQPInput {
	conn, err := amqp.Dial(addr)
	if err != nil {
		log.Fatal(err)
	}

	return AMQPInput{conn: conn}
}

func (in AMQPInput) GetMetric(name string) (int, error) {
	channel, err := in.conn.Channel()
	if err != nil {
		return 0, err
	}

	defer channel.Close()
	queue, err := channel.QueueInspect(name)
	if err != nil {
		return 0, err
	}

	return queue.Messages, nil
}
