package config

import "github.com/spf13/viper"

type App struct {
	AppPort string `json:"app_port"`
	AppEnv  string `json:"app_env"`

	JwtSecretKey string `json:"jwt_secret_key"`

	ServerTimeOut     int    `json:"server_timeout"`
	ProductServiceUrl string `json:"product_service_url"`
	UserServiceUrl    string `json:"user_service_url"`
	OrderServiceUrl   string `json:"order_service_url"`
}

type PsqlDB struct {
	Host      string `json:"host"`
	Port      string `json:"port"`
	User      string `json:"user"`
	Password  string `json:"password"`
	DBName    string `json:"db_name"`
	DBMaxOpen int    `json:"db_max_open"`
	DBMaxIdle int    `json:"db_max_idle"`
}

type RabbitMQ struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`

	PublisherPoolSize   int `json:"publisher_pool_size"`
	PublisherMaxRetries int `json:"publisher_max_retries"`
	PublishTimeoutSec   int `json:"publish_timeout_sec"`
}

type Redis struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

type Midtrans struct {
	ServerKey   string `json:"server_key"`
	Environment int    `json:"environment"`
}

type PublisherName struct {
	PaymentSuccess string `json:"payment_success"`
}

type Config struct {
	App           App           `json:"app"`
	Psql          PsqlDB        `json:"psql"`
	RabbitMQ      RabbitMQ      `json:"rabbitmq"`
	Redis         Redis         `json:"redis"`
	Midtrans      Midtrans      `json:"midtrans"`
	PublisherName PublisherName `json:"publisher_name"`
}

func NewConfig() *Config {
	return &Config{
		App: App{
			AppPort: viper.GetString("APP_PORT"),
			AppEnv:  viper.GetString("APP_PORT"),

			JwtSecretKey:      viper.GetString("JWT_SECRET_KEY"),
			ServerTimeOut:     viper.GetInt("SERVER_TIMEOUT"),
			ProductServiceUrl: viper.GetString("PRODUCT_SERVICE_URL"),
			UserServiceUrl:    viper.GetString("USER_SERVICE_URL"),
			OrderServiceUrl:   viper.GetString("ORDER_SERVICE_URL"),
		},
		Psql: PsqlDB{
			Host:      viper.GetString("DATABASE_HOST"),
			Port:      viper.GetString("DATABASE_PORT"),
			User:      viper.GetString("DATABASE_USER"),
			Password:  viper.GetString("DATABASE_PASSWORD"),
			DBName:    viper.GetString("DATABASE_NAME"),
			DBMaxOpen: viper.GetInt("DATABASE_MAX_OPEN_CONNECTION"),
			DBMaxIdle: viper.GetInt("DATABASE_MAX_IDLE_CONNECTION"),
		},
		RabbitMQ: RabbitMQ{
			Host:                viper.GetString("RABBITMQ_HOST"),
			Port:                viper.GetString("RABBITMQ_PORT"),
			User:                viper.GetString("RABBITMQ_USER"),
			Password:            viper.GetString("RABBITMQ_PASSWORD"),
			PublisherPoolSize:   viper.GetInt("RABBITMQ_PUBLISHER_POOL_SIZE"),
			PublisherMaxRetries: viper.GetInt("RABBITMQ_PUBLISHER_MAX_RETRIES"),
			PublishTimeoutSec:   viper.GetInt("RABBITMQ_PUBLISH_TIMEOUT_SEC"),
		},
		Redis: Redis{
			Host: viper.GetString("REDIS_HOST"),
			Port: viper.GetString("REDIS_PORT"),
		},
		Midtrans: Midtrans{
			ServerKey:   viper.GetString("MIDTRANS_SERVER_KEY"),
			Environment: viper.GetInt("MIDTRANS_ENVIRONMENT"),
		},
		PublisherName: PublisherName{
			PaymentSuccess: viper.GetString("PUBLISHER_PAYMENT_SUCCESS"),
		},
	}
}
