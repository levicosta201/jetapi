package sites

import (
	"context"
	"fmt"
	"time"

	"github.com/macsencasaus/jetapi/internal/cache"
	"golang.org/x/sync/errgroup"
)

type ScrapeResult struct {
	JetPhotos   *JetPhotosResult
	FlightRadar *FlightRadarResult
}

type APIQueries struct {
	Reg     string
	Photos  int
	Flights int
	OnlyJP  bool
	OnlyFR  bool
}

var cacheManager *cache.Manager

// SetCacheManager configura o gerenciador de cache
func SetCacheManager(cm *cache.Manager) {
	cacheManager = cm
}

func Scrape(q *APIQueries) (*ScrapeResult, error) {
	// Tentar recuperar do cache primeiro
	if cacheManager != nil && cacheManager.IsCacheAvailable() {
		cacheKey := cacheManager.GetCache().GenerateKey("scrape", q)
		var cachedResult *ScrapeResult
		found, err := cacheManager.GetCache().Get(cacheKey, &cachedResult)
		if err == nil && found && cachedResult != nil {
			// Garantir que estruturas não sejam nil
			if cachedResult.JetPhotos == nil {
				cachedResult.JetPhotos = &JetPhotosResult{Images: []ImageAttributes{}}
			}
			if cachedResult.FlightRadar == nil {
				cachedResult.FlightRadar = &FlightRadarResult{Flights: []*FlightAttributes{}}
			}
			return cachedResult, nil
		}
	}
	var (
		jpResult *JetPhotosResult
		frResult *FlightRadarResult
		jpErr    error
		frErr    error
	)

	g, _ := errgroup.WithContext(context.Background())

	g.Go(func() error {
		res, err := ScrapeJetPhotos(q)
		if err != nil {
			jpErr = err
			return nil // Não retornar erro para permitir que FlightRadar tente
		}
		jpResult = res
		return nil
	})

	g.Go(func() error {
		res, err := ScrapeFlightRadar(q)
		if err != nil {
			frErr = err
			return nil // Não retornar erro para permitir que JetPhotos tente
		}
		frResult = res
		return nil
	})

	_ = g.Wait()

	// Garantir que sempre temos estruturas inicializadas, mesmo se uma parte falhar
	if jpResult == nil {
		jpResult = &JetPhotosResult{Images: []ImageAttributes{}}
	}
	if frResult == nil {
		frResult = &FlightRadarResult{Flights: []*FlightAttributes{}}
	}

	// Se ambos falharam, retornar erro combinado
	if jpErr != nil && frErr != nil {
		return nil, fmt.Errorf("JetPhotos Error: %v; FlightRadar Error: %v", jpErr, frErr)
	}

	result := &ScrapeResult{JetPhotos: jpResult, FlightRadar: frResult}

	// Armazenar no cache se disponível
	if cacheManager != nil && cacheManager.IsCacheAvailable() {
		cacheKey := cacheManager.GetCache().GenerateKey("scrape", q)
		// Cachear por 24 horas para resultados completos
		cacheManager.GetCache().SetWithTTL(cacheKey, result, 24*time.Hour)
	}

	// Se chegou até aqui, pelo menos um scraper funcionou, então retornar nil como erro
	// (erros parciais são ignorados se pelo menos um resultado foi obtido)
	return result, nil
}
