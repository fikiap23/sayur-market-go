package config

import "github.com/spf13/viper"

type App struct {
	AppPort string `json:"app_port"`
	AppEnv  string `json:"app_env"`

	JwtSecretKey string `json:"jwt_secret_key"`
	JwtIssuer    string `json:"jwt_issuer"`

	UrlForgotPassword string `json:"url_forgot_password"`
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
}

type Supabase struct {
	URL    string `json:"url"`
	Key    string `json:"key"`
	Bucket string `json:"bucket"`
}

type Redis struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

type ElasticSearch struct {
	Host string `json:"host"`
}

type PublisherName struct {
	ProductUpdateStock string `json:"product_update_stock"`
	ProductPublish     string `json:"product_publish"`
	ProductDelete      string `json:"product_delete"`
	ProductToOrder     string `json:"product_to_order"`
}

type Config struct {
	App           App           `json:"app"`
	Psql          PsqlDB        `json:"psql"`
	RabbitMQ      RabbitMQ      `json:"rabbitmq"`
	Storage       Supabase      `json:"storage"`
	Redis         Redis         `json:"redis"`
	ElasticSearch ElasticSearch `json:"elasticsearch"`
	PublisherName PublisherName `json:"publisher_name"`
}

func NewConfig() *Config {
	return &Config{
		App: App{
			AppPort: viper.GetString("APP_PORT"),
			AppEnv:  viper.GetString("APP_PORT"),

			JwtSecretKey: viper.GetString("JWT_SECRET_KEY"),
			JwtIssuer:    viper.GetString("JWT_ISSUER"),

			UrlForgotPassword: viper.GetString("URL_FORGOT_PASSWORD"),
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
			Host:     viper.GetString("RABBITMQ_HOST"),
			Port:     viper.GetString("RABBITMQ_PORT"),
			User:     viper.GetString("RABBITMQ_USER"),
			Password: viper.GetString("RABBITMQ_PASSWORD"),
		},
		Storage: Supabase{
			URL:    viper.GetString("SUPABASE_STORAGE_URL"),
			Key:    viper.GetString("SUPABASE_STORAGE_KEY"),
			Bucket: viper.GetString("SUPABASE_STORAGE_BUCKET"),
		},
		Redis: Redis{
			Host: viper.GetString("REDIS_HOST"),
			Port: viper.GetString("REDIS_PORT"),
		},
		ElasticSearch: ElasticSearch{
			Host: viper.GetString("ELASTICSEARCH_HOST"),
		},
		PublisherName: PublisherName{
			ProductUpdateStock: viper.GetString("PRODUCT_UPDATE_STOCK_NAME"),
			ProductPublish:     viper.GetString("PRODUCT_PUBLISH_NAME"),
			ProductDelete:      viper.GetString("PRODUCT_DELETE"),
			ProductToOrder:     viper.GetString("PRODUCT_TO_ORDER"),
		},
	}
}
