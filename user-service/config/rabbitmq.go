package config

import "fmt"

// RabbitMQURL builds the AMQP connection string from config fields.
func (cfg Config) RabbitMQURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/",
		cfg.RabbitMQ.User, cfg.RabbitMQ.Password,
		cfg.RabbitMQ.Host, cfg.RabbitMQ.Port,
	)
}
