package scraper

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
)

// ProxyManager gerencia múltiplos proxies com rotação
type ProxyManager struct {
	proxies []string
	current int
	mu      sync.Mutex
}

var globalProxyManager *ProxyManager
var proxyManagerOnce sync.Once

// GetProxyManager retorna o gerenciador de proxies global (singleton)
func GetProxyManager() *ProxyManager {
	proxyManagerOnce.Do(func() {
		globalProxyManager = NewProxyManager()
	})
	return globalProxyManager
}

// NewProxyManager cria um novo gerenciador de proxies
func NewProxyManager() *ProxyManager {
	pm := &ProxyManager{
		proxies: []string{},
		current: 0,
	}

	// Carregar proxies da variável de ambiente
	proxyURLs := os.Getenv("PROXY_URLS")
	if proxyURLs != "" {
		// Suporta múltiplos formatos: vírgula, ponto e vírgula, ou quebra de linha
		separators := []string{",", ";", "\n", " "}
		proxies := []string{proxyURLs}
		
		for _, sep := range separators {
			var newProxies []string
			for _, p := range proxies {
				parts := strings.Split(p, sep)
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if part != "" {
						newProxies = append(newProxies, part)
					}
				}
			}
			proxies = newProxies
		}
		
		pm.proxies = proxies
		log.Printf("[ProxyManager] Carregados %d proxies da variável de ambiente", len(pm.proxies))
		
		// Log apenas os primeiros 5 e últimos 5 proxies para não poluir os logs
		if len(pm.proxies) > 10 {
			log.Printf("[ProxyManager] Exibindo primeiros e últimos proxies como exemplo:")
			for i := 0; i < 5 && i < len(pm.proxies); i++ {
				masked := maskProxyURL(pm.proxies[i])
				log.Printf("[ProxyManager] Proxy %d: %s", i+1, masked)
			}
			log.Printf("[ProxyManager] ... (%d proxies no meio) ...", len(pm.proxies)-10)
			for i := len(pm.proxies) - 5; i < len(pm.proxies); i++ {
				masked := maskProxyURL(pm.proxies[i])
				log.Printf("[ProxyManager] Proxy %d: %s", i+1, masked)
			}
		} else {
			// Se tiver poucos proxies, mostrar todos
			for i, p := range pm.proxies {
				masked := maskProxyURL(p)
				log.Printf("[ProxyManager] Proxy %d: %s", i+1, masked)
			}
		}
	} else {
		log.Printf("[ProxyManager] Nenhum proxy configurado (PROXY_URLS não definido)")
	}

	return pm
}

// maskProxyURL mascarar credenciais na URL do proxy para logs
// Retorna apenas host:porta sem credenciais
func maskProxyURL(proxyURL string) string {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return "***"
	}
	// Retornar apenas o esquema, host e porta (sem credenciais)
	if u.Host != "" {
		return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	}
	return "***"
}

// GetNextProxy retorna o próximo proxy na rotação (round-robin)
func (pm *ProxyManager) GetNextProxy() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return ""
	}

	currentIdx := pm.current
	proxy := pm.proxies[currentIdx]
	pm.current = (pm.current + 1) % len(pm.proxies)
	
	masked := maskProxyURL(proxy)
	log.Printf("[ProxyManager] Usando proxy (índice %d/%d): %s", currentIdx+1, len(pm.proxies), masked)
	
	return proxy
}

// GetAllProxies retorna todos os proxies disponíveis
func (pm *ProxyManager) GetAllProxies() []string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Retornar cópia para evitar race conditions
	proxies := make([]string, len(pm.proxies))
	copy(proxies, pm.proxies)
	return proxies
}

// HasProxies retorna true se há proxies configurados
func (pm *ProxyManager) HasProxies() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.proxies) > 0
}


