package app

import (
	"fmt"
	"log"

	"github.com/ternaryinvalid/balancer/internal/balancer"
	"github.com/ternaryinvalid/balancer/internal/config"
	db "github.com/ternaryinvalid/balancer/internal/database"
	"github.com/ternaryinvalid/balancer/internal/router"
)

func Start(cfg *config.Config) {
	var backendConfigs []struct{ Host, Port string }
	for _, back := range cfg.Backends {
		go router.StartBackend(back.Host, back.Port)
		backendConfigs = append(backendConfigs, struct{ Host, Port string }{
			Host: back.Host,
			Port: back.Port,
		})
	}

	pool := balancer.NewServerPool(backendConfigs)

	database, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("Не получается подключиться к БД: %v", err)
	}
	defer database.Close()

	if err := database.CreateRateLimitTable(); err != nil {
		log.Fatalf("Не получается создать БД: %v", err)
	}

	fmt.Println("Успешное подключение к БД и инициализация таблиц")

	go balancer.StartHealthCheck(pool)
	router.StartBalancer(cfg.HTTP.Host, cfg.HTTP.Port, pool, database)
}
