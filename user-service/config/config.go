package config

import (
	"strings"

	"github.com/spf13/viper"
)

type App struct {
	AppPort string `json:"app_port"`
	AppEnv  string `json:"app_env"`

	JwtSecretKey string `json:"jwt_secret_key"`
	JwtIssuer    string `json:"jwt_issuer"`

	UrlForgotPassword string `json:"url_forgot_password"`
	UrlFrontFE        string `json:"url_front_fe"`
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



type Config struct {
	App      App      `json:"app"`
	Psql     PsqlDB   `json:"psql"`
}

func NewConfig() *Config {
	return &Config{
		App: App{
			AppPort: viper.GetString("APP_PORT"),
			AppEnv:  viper.GetString("APP_ENV"),

			JwtSecretKey: viper.GetString("JWT_SECRET_KEY"),
			JwtIssuer:    viper.GetString("JWT_ISSUER"),

			UrlForgotPassword: viper.GetString("URL_FORGOT_PASSWORD"),
			UrlFrontFE:        viper.GetString("URL_FRONT_FE"),
		},
		Psql: PsqlDB{
			Host:      strings.TrimSpace(viper.GetString("DATABASE_HOST")),
			Port:      strings.TrimSpace(viper.GetString("DATABASE_PORT")),
			User:      strings.TrimSpace(viper.GetString("DATABASE_USER")),
			Password:  strings.TrimSpace(viper.GetString("DATABASE_PASSWORD")),
			DBName:    strings.TrimSpace(viper.GetString("DATABASE_NAME")),
			DBMaxOpen: viper.GetInt("DATABASE_MAX_OPEN_CONNECTION"),
			DBMaxIdle: viper.GetInt("DATABASE_MAX_IDLE_CONNECTION"),
		},
	}
}
