package config

import (
	"log"

	"github.com/spf13/viper"
)

type Env struct {
	DatabaseUrl string `mapstructure:"DATABASE_URL"`
	WebhookUrl  string `mapstructure:"WEBHOOK_URL"`
}

func NewEnv(filename string, override bool) *Env {

	env := Env{}
	viper.SetConfigFile(filename)

	if override {
		viper.AutomaticEnv()
	}

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("Error reading environment file: ", err)
	}

	if err := viper.Unmarshal(&env); err != nil {
		log.Fatal("Error loading environment file")
	}

	return &env
}
