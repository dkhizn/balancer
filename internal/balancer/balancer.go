package balancer

import (
	"log"
	"net"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// Реализован round robin

type Backend struct {
	URL          *url.URL
	ReverseProxy *httputil.ReverseProxy
	Alive        bool
	mu           sync.RWMutex
}

type ServerPool struct {
	backends []*Backend
	index    int
	mu       sync.RWMutex
}

func NewServerPool(backendConfigs []struct{ Host, Port string }) *ServerPool {
	var backends []*Backend
	for _, back := range backendConfigs {
		urlStr := "http://" + back.Host + ":" + back.Port
		url, err := url.Parse(urlStr)
		if err != nil {
			log.Printf("Неправильный url бэкенда из конфига %q: %v", urlStr, err)
			continue
		}

		backends = append(backends, &Backend{
			URL:          url,
			ReverseProxy: httputil.NewSingleHostReverseProxy(url),
			Alive:        true,
		})
	}
	return &ServerPool{backends: backends}
}

func (p *ServerPool) Next() *Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()

	next := p.index
	// Цикл нужен, чтобы учесть, что текущий бэкенд может не отвечать
	// Тогда нужно циклом пройтись по всем следующим бэкендам
	for i := 0; i < len(p.backends); i++ {
		next = (next + 1) % len(p.backends)
		if p.backends[next].Alive {
			p.index = next
			log.Printf("Запрос перенаправлен на бэкенд: %s", p.backends[next].URL.String())

			return p.backends[next]
		} else {
			log.Printf("Невозможно направить запрос на: %s", p.backends[next].URL.String())
			log.Printf("Пробуем отправить запрос на следующий бэкенд")
		}
	}
	return nil
}

func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	b.Alive = alive
	b.mu.Unlock()
}

func (b *Backend) IsAlive() (alive bool) {
	b.mu.RLock()
	alive = b.Alive
	b.mu.RUnlock()
	return
}

func (p *ServerPool) healthCheck() {
	for _, back := range p.backends {
		status := checkAlive(back.URL)
		back.SetAlive(status)
	}
}

func checkAlive(u *url.URL) bool {
	timeout := 3 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Printf("Бэкенд недостижим, %v", err)

		return false
	}

	_ = conn.Close()
	return true

}

func StartHealthCheck(pool *ServerPool) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Запущена проверка бэкендов")
		pool.healthCheck()
	}
}
