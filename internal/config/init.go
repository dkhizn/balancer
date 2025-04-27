package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

func New() (*Config, error) {
	cfg := &Config{}
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "./config.yml"
	}
	err := cleanenv.ReadConfig(path, cfg)
	if err != nil {
		log.Println("Ошибка при чтении конфига")
		return nil, err
	}
	return cfg, nil
}
