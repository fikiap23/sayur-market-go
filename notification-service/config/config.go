package config

import "github.com/spf13/viper"

type App struct {
	AppPort string `json:"app_port"`
	AppEnv  string `json:"app_env"`

	JwtSecretKey string `json:"jwt_secret_key"`
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

type Redis struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

type RabbitMQ struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type EmailConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Sending  string `json:"sending"`
	IsTLS    bool   `json:"is_tls"`
}

type Config struct {
	App       App         `json:"app"`
	Psql      PsqlDB      `json:"psql"`
	RabbitMQ  RabbitMQ    `json:"rabbitmq"`
	EmailConf EmailConfig `json:"email_conf"`
	Redis     Redis       `json:"redis"`
}

func NewConfig() *Config {
	return &Config{
		App: App{
			AppPort: viper.GetString("APP_PORT"),
			AppEnv:  viper.GetString("APP_PORT"),

			JwtSecretKey: viper.GetString("JWT_SECRET_KEY"),
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
		EmailConf: EmailConfig{
			Host:     viper.GetString("EMAIL_HOST"),
			Port:     viper.GetInt("EMAIL_PORT"),
			Username: viper.GetString("EMAIL_USERNAME"),
			Password: viper.GetString("EMAIL_PASSWORD"),
			Sending:  viper.GetString("EMAIL_SENDING"),
			IsTLS:    viper.GetBool("EMAIL_IS_TLS"),
		},
		Redis: Redis{
			Host: viper.GetString("REDIS_HOST"),
			Port: viper.GetString("REDIS_PORT"),
		},
	}
}
