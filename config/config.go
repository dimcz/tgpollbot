package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/sirupsen/logrus"
)

type config struct {
	Port    int    `env:"PORT" env-default:"8080"`
	XApiKey string `env:"X_API_KEY" env-required:"true"`
	RedisDB string `env:"REDIS_DB" env-required:"true"`
	Users   string `env:"USERS" env-required:"true"`
	Token   string `env:"TOKEN" env-required:"true"`
}

var Config config

func init() {
	if err := cleanenv.ReadEnv(&Config); err != nil {
		logrus.Fatal(err)
	}
}
