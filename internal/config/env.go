package config

import (
	"log"

	"github.com/spf13/viper"
)

type Env struct {
	DatabaseUrl           string `mapstructure:"DATABASE_URL"`
	WebhookUrl            string `mapstructure:"WEBHOOK_URL"`
	WasabiAccessKeyID     string `mapstructure:"WASABI_ACCESS_KEY_ID"`
	WasabiSecretAccessKey string `mapstructure:"WASABI_SECRET_ACCESS_KEY"`
	WasabiEndpoint        string `mapstructure:"WASABI_ENDPOINT"`
	WasabiBucket          string `mapstructure:"WASABI_BUCKET"`
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
