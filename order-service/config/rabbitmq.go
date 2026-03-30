package config

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQURL builds the AMQP connection string from config fields.
func (cfg Config) RabbitMQURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/",
		cfg.RabbitMQ.User, cfg.RabbitMQ.Password,
		cfg.RabbitMQ.Host, cfg.RabbitMQ.Port,
	)
}

// DialRabbitMQ opens a single AMQP connection (used by standalone worker commands).
func (cfg Config) DialRabbitMQ() (*amqp.Connection, error) {
	return amqp.Dial(cfg.RabbitMQURL())
}
