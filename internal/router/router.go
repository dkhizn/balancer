package router

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ternaryinvalid/balancer/internal/balancer"
	db "github.com/ternaryinvalid/balancer/internal/database"
	ratelimit "github.com/ternaryinvalid/balancer/internal/ratelimiter"
)

func StartBackend(host string, port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", BackHandler(port))
	url := host + ":" + port

	server := &http.Server{
		Addr:    url,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	go func() {
		log.Printf("Бэкенд %s запущен", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Бэкенд %s упал: %v", port, err)
		}
	}()

	<-ctx.Done()
	log.Printf("Graceful shutdown бэкенда %s", port)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при остановке бэкенда %s: %v", port, err)
	}
}

func StartBalancer(host string, port string, pool *balancer.ServerPool, db *db.DB) {
	mux := http.NewServeMux()
	rl := ratelimit.NewRateLimiter(db)
	mux.HandleFunc("/", LoadBalancerHandler(pool, rl))
	mux.HandleFunc("/ratelimit", SetRateLimitHandler(rl))
	url := host + ":" + port

	server := &http.Server{
		Addr:    url,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	go func() {
		log.Printf("Балансер запущен")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Балансер упал: %v", err)
		}
	}()

	<-ctx.Done()
	log.Print("Graceful shutdown балансера")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при остановке балансера: %v", err)
	}
}
