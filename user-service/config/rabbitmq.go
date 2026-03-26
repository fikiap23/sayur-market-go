package config

import (
	"fmt"

	"github.com/streadway/amqp"
)

func (cfg Config) NewRabbitMQ() (*amqp.Connection, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.RabbitMQ.User, cfg.RabbitMQ.Password, cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	conn, err := amqp.Dial(url)
	if err != nil {
		fmt.Printf("[NewRabbitMQ-1] Failed to connect to RabbitMQ: %v", err)
		return nil, err
	}

	return conn, nil
}
