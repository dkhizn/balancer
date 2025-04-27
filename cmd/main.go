package main

import (
	"log"

	"github.com/ternaryinvalid/balancer/internal/app"
	"github.com/ternaryinvalid/balancer/internal/config"
)

func main() {
	cfg, err := config.New()

	if err != nil {
		log.Fatalf("Error at start: %v", err)
	}

	app.Start(cfg)
}
