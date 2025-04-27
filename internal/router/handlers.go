package router

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/ternaryinvalid/balancer/internal/balancer"
	ratelimit "github.com/ternaryinvalid/balancer/internal/ratelimiter"
)

func BackHandler(port string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Это сервер на порту %s", port)
	}
}

func LoadBalancerHandler(pool *balancer.ServerPool, rl *ratelimit.RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1) header → query param → HTML-форма
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("client_id")
		}

		// 2) rate-limit
		if !rl.Allow(apiKey) {
			http.Error(w, "Слишком много запросов", http.StatusTooManyRequests)
			return
		}

		// 3) балансинг
		server := pool.Next()
		if server == nil {
			log.Printf("Нет живых бэкендов")
			http.Error(w, "Нет живых бэкендов", http.StatusServiceUnavailable)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(server.URL)
		proxy.ServeHTTP(w, r)
	}
}

func SetRateLimitHandler(rl *ratelimit.RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Разрешаем как GET, так и POST
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
			return
		}

		// Берём параметры из URL (для GET) или формы (для POST)
		clientID := r.URL.Query().Get("client_id")
		capacityStr := r.URL.Query().Get("capacity")
		rateStr := r.URL.Query().Get("rate")

		if clientID == "" || capacityStr == "" || rateStr == "" {
			http.Error(w, "Отсутствуют параметры", http.StatusBadRequest)
			return
		}

		capacity, err := strconv.Atoi(capacityStr)
		if err != nil {
			http.Error(w, "Некорректная ёмкость", http.StatusBadRequest)
			return
		}

		rate, err := strconv.Atoi(rateStr)
		if err != nil {
			http.Error(w, "Некорректная скорость", http.StatusBadRequest)
			return
		}

		err = rl.SetRule(clientID, capacity, rate)
		if err != nil {
			http.Error(w, fmt.Sprintf("Не удалось установить лимит: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Лимит обновлён для клиента %s", clientID)
	}
}
