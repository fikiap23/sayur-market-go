package config

import "github.com/spf13/viper"

type App struct {
	AppPort string `json:"app_port"`
	AppEnv  string `json:"app_env"`

	JwtSecretKey string `json:"jwt_secret_key"`

	ServerTimeOut     int    `json:"server_timeout"`
	ProductServiceUrl string `json:"product_service_url"`
	UserServiceUrl    string `json:"user_service_url"`

	LatitudeRef  string `json:"latitude_ref"`
	LongitudeRef string `json:"longitude_ref"`
	MaxDistance  int    `json:"max_distance"`
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

	WorkerPoolSize    int `json:"worker_pool_size"`
	PrefetchCount     int `json:"prefetch_count"`
	MaxRetries        int `json:"max_retries"`
	ProcessTimeoutSec int `json:"process_timeout_sec"`
}

type Redis struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

type PublisherName struct {
	ProductUpdateStock      string `json:"product_update_stock"`
	OrderPublish            string `json:"order_publish"`
	EmailUpdateStatus       string `json:"email_update_status"`
	PublisherDeleteOrder    string `json:"publisher_delete_order"`
	PublisherPaymentSuccess string `json:"publisher_payment_success"`
	PublisherUpdateStatus   string `json:"publisher_update_status"`
}

type ElasticSearch struct {
	Host string `json:"host"`
}

type Config struct {
	App           App           `json:"app"`
	Psql          PsqlDB        `json:"psql"`
	RabbitMQ      RabbitMQ      `json:"rabbitmq"`
	Redis         Redis         `json:"redis"`
	PublisherName PublisherName `json:"publisher_name"`
	ElasticSearch ElasticSearch `json:"elasticsearch"`
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
			LatitudeRef:       viper.GetString("LATITUDE_REF"),
			LongitudeRef:      viper.GetString("LONGITUDE_REF"),
			MaxDistance:       viper.GetInt("MAX_DISTANCE"),
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
			WorkerPoolSize:      viper.GetInt("RABBITMQ_WORKER_POOL_SIZE"),
			PrefetchCount:       viper.GetInt("RABBITMQ_PREFETCH_COUNT"),
			MaxRetries:          viper.GetInt("RABBITMQ_MAX_RETRIES"),
			ProcessTimeoutSec:   viper.GetInt("RABBITMQ_PROCESS_TIMEOUT_SEC"),
		},
		Redis: Redis{
			Host: viper.GetString("REDIS_HOST"),
			Port: viper.GetString("REDIS_PORT"),
		},
		PublisherName: PublisherName{
			ProductUpdateStock:      viper.GetString("PRODUCT_UPDATE_STOCK_NAME"),
			OrderPublish:            viper.GetString("ORDER_PUBLISH_NAME"),
			EmailUpdateStatus:       viper.GetString("EMAIL_UPDATE_STATUS_NAME"),
			PublisherDeleteOrder:    viper.GetString("PUBLISHER_DELETE_ORDER"),
			PublisherPaymentSuccess: viper.GetString("PUBLISHER_PAYMENT_SUCCESS"),
			PublisherUpdateStatus:   viper.GetString("PUBLISHER_UPDATE_STATUS"),
		},
		ElasticSearch: ElasticSearch{
			Host: viper.GetString("ELASTICSEARCH_HOST"),
		},
	}
}
