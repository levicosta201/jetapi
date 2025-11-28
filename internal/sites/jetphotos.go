package sites

import (
	"fmt"
	"log"
	"strings"

	"context"
	"github.com/macsencasaus/jetapi/internal/scraper"
	"golang.org/x/sync/errgroup"
)

type JetPhotosResult struct {
	Reg    string            `json:"Reg"`
	Images []ImageAttributes `json:"Images"`
}

type ImageAttributes struct {
	Image        string `json:"Image"`
	Link         string `json:"Link"`
	Thumbnail    string `json:"Thumbnail"`
	DateTaken    string `json:"DateTaken"`
	DateUploaded string `json:"DateUploaded"`
	Location     string `json:"Location"`
	Photographer string `json:"Photographer"`
	Aircraft     string `json:"Aircraft"`
	Serial       string `json:"Serial"`
	Airline      string `json:"Airline"`
}

const jpHomeURL = "https://www.jetphotos.com"

func ScrapeJetPhotos(q *APIQueries) (*JetPhotosResult, error) {
	reg := q.Reg
	log.Printf("[JetPhotos] Iniciando busca para registro: %s, fotos solicitadas: %d", reg, q.Photos)
	
	if q.Photos == 0 {
		log.Printf("[JetPhotos] Nenhuma foto solicitada para %s, retornando resultado vazio", reg)
		return &JetPhotosResult{Reg: strings.ToUpper(reg)}, nil
	}

	URL := fmt.Sprintf("%s/photo/keyword/%s", jpHomeURL, reg)
	log.Printf("[JetPhotos] Buscando URL: %s", URL)
	b, err := scraper.FetchHTML(URL)
	if err != nil {
		log.Printf("[JetPhotos] ERRO ao buscar HTML da URL %s: %v", URL, err)
		return nil, jpError("scraping search URL", reg, URL, err)
	}
	log.Printf("[JetPhotos] HTML obtido com sucesso da URL: %s", URL)

	s := scraper.NewScraper(b)
	defer s.Close()

	pageLinks := []string{}
	thumbnails := []string{}
	log.Printf("[JetPhotos] Iniciando busca de pageLinks e thumbnails (tentando encontrar %d)", q.Photos)
	for i := 0; i < q.Photos; i++ {
		log.Printf("[JetPhotos] Tentativa %d/%d: buscando pageLink...", i+1, q.Photos)
		pageLink, err := s.ScrapeLinks("a", "result__photoLink", 1)
		if err != nil {
			log.Printf("[JetPhotos] ERRO ao buscar pageLink %d: %v (já encontrados: %d)", i+1, err, len(pageLinks))
			if len(pageLinks) > 0 {
				log.Printf("[JetPhotos] Continuando com %d pageLinks encontrados", len(pageLinks))
				break
			}
			return nil, jpError("scraping aircraft pagelinks", reg, URL, err)
		}
		if len(pageLink) == 0 || pageLink[0] == "" {
			log.Printf("[JetPhotos] AVISO: pageLink %d está vazio (já encontrados: %d)", i+1, len(pageLinks))
			if len(pageLinks) > 0 {
				log.Printf("[JetPhotos] Continuando com %d pageLinks encontrados", len(pageLinks))
				break
			}
			return nil, jpError("scraping aircraft pagelinks", reg, URL, fmt.Errorf("empty pageLink found"))
		}
		log.Printf("[JetPhotos] PageLink %d encontrado: %s", i+1, pageLink[0])

		log.Printf("[JetPhotos] Tentativa %d/%d: buscando thumbnail...", i+1, q.Photos)
		thumbnail, err := s.ScrapeLinks("img", "result__photo", 1)
		if err != nil {
			log.Printf("[JetPhotos] ERRO ao buscar thumbnail %d: %v (já encontrados: %d)", i+1, err, len(thumbnails))
			if len(thumbnails) > 0 {
				log.Printf("[JetPhotos] Continuando com %d thumbnails encontrados", len(thumbnails))
				break
			}
			return nil, jpError("scraping aircraft thumbnails", reg, URL, err)
		}
		if len(thumbnail) == 0 || thumbnail[0] == "" {
			log.Printf("[JetPhotos] AVISO: thumbnail %d está vazio (já encontrados: %d)", i+1, len(thumbnails))
			if len(thumbnails) > 0 {
				log.Printf("[JetPhotos] Continuando com %d thumbnails encontrados", len(thumbnails))
				break
			}
			return nil, jpError("scraping aircraft thumbnails", reg, URL, fmt.Errorf("empty thumbnail found"))
		}
		log.Printf("[JetPhotos] Thumbnail %d encontrado: %s", i+1, thumbnail[0])
		pageLinks = append(pageLinks, pageLink[0])
		thumbnails = append(thumbnails, thumbnail[0])
	}
	log.Printf("[JetPhotos] Total encontrado: %d pageLinks, %d thumbnails", len(pageLinks), len(thumbnails))
	
	if len(pageLinks) == 0 {
		log.Printf("[JetPhotos] ERRO: Nenhuma imagem encontrada para registro %s", reg)
		return nil, jpError("no images found", reg, URL, fmt.Errorf("no images found for registration %s", reg))
	}

	images := make([]ImageAttributes, len(pageLinks))
	log.Printf("[JetPhotos] Iniciando processamento de %d imagens em paralelo", len(pageLinks))

	pageScraper := func(i int, link string) error {
		photoURL := fmt.Sprintf("%s%s", jpHomeURL, link)
		log.Printf("[JetPhotos] [Imagem %d/%d] Processando: %s", i+1, len(pageLinks), photoURL)
		images[i].Link = photoURL
		
		thumbnailURL := "https:" + thumbnails[i]
		log.Printf("[JetPhotos] [Imagem %d/%d] Thumbnail: %s", i+1, len(pageLinks), thumbnailURL)
		images[i].Thumbnail = thumbnailURL

		log.Printf("[JetPhotos] [Imagem %d/%d] Buscando HTML da página da foto: %s", i+1, len(pageLinks), photoURL)
		b, err := scraper.FetchHTML(photoURL)
		if err != nil {
			log.Printf("[JetPhotos] [Imagem %d/%d] ERRO ao buscar HTML: %v", i+1, len(pageLinks), err)
			return jpError("fetching HTML page", reg, URL, err)
		}
		log.Printf("[JetPhotos] [Imagem %d/%d] HTML obtido com sucesso", i+1, len(pageLinks))

		s := scraper.NewScraper(b)
		defer s.Close()

		// photo links
		log.Printf("[JetPhotos] [Imagem %d/%d] Buscando link da foto grande...", i+1, len(pageLinks))
		photoLinkArr, err := s.ScrapeLinks("img", "large-photo__img", 1)
		if err != nil {
			log.Printf("[JetPhotos] [Imagem %d/%d] ERRO ao buscar foto grande: %v", i+1, len(pageLinks), err)
			return jpError("scraping photo links", reg, URL, err)
		}
		if len(photoLinkArr) == 0 || photoLinkArr[0] == "" {
			log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: foto grande está vazia", i+1, len(pageLinks))
			return jpError("scraping photo links", reg, URL, fmt.Errorf("empty photo link found"))
		}
		log.Printf("[JetPhotos] [Imagem %d/%d] Foto grande encontrada: %s", i+1, len(pageLinks), photoLinkArr[0])
		images[i].Image = photoLinkArr[0]

		// registration + dates (opcional - não retorna erro se não encontrar)
		log.Printf("[JetPhotos] [Imagem %d/%d] Buscando registro e datas...", i+1, len(pageLinks))
		res, err := s.ScrapeText("h4", "headerText4 color-shark", 3)
		if err != nil {
			log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: Não foi possível buscar datas: %v (continuando sem datas)", i+1, len(pageLinks), err)
		} else if len(res) >= 3 {
			images[i].DateTaken = res[1]
			images[i].DateUploaded = res[2]
			log.Printf("[JetPhotos] [Imagem %d/%d] Data tirada: %s, Data upload: %s", i+1, len(pageLinks), res[1], res[2])
		} else {
			log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: Menos de 3 resultados para datas. Encontrado: %v", i+1, len(pageLinks), res)
		}

		// aircraft (opcional - não retorna erro se não encontrar)
		log.Printf("[JetPhotos] [Imagem %d/%d] Buscando informações da aeronave...", i+1, len(pageLinks))
		err = s.Advance("h2", "header-reset", 1)
		if err != nil {
			log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: Não foi possível avançar para seção de aeronave: %v (continuando sem informações de aeronave)", i+1, len(pageLinks), err)
		} else {
			res, err = s.ScrapeText("a", "link", 3)
			if err != nil {
				log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: Não foi possível buscar informações de aeronave: %v (continuando sem informações)", i+1, len(pageLinks), err)
			} else if len(res) >= 3 {
				images[i].Aircraft = res[0]
				images[i].Airline = res[1]
				images[i].Serial = strings.TrimSpace(res[2])
				log.Printf("[JetPhotos] [Imagem %d/%d] Aeronave: %s, Companhia: %s, Serial: %s", i+1, len(pageLinks), res[0], res[1], res[2])
			} else {
				log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: Menos de 3 resultados para aeronave. Encontrado: %v", i+1, len(pageLinks), res)
			}
		}

		// location (opcional - não retorna erro se não encontrar)
		log.Printf("[JetPhotos] [Imagem %d/%d] Buscando localização...", i+1, len(pageLinks))
		err = s.Advance("h5", "header-reset", 1)
		if err != nil {
			log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: Não foi possível avançar para seção de localização: %v (continuando sem localização)", i+1, len(pageLinks), err)
		} else {
			location, err := s.ScrapeText("a", "link", 1)
			if err != nil {
				log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: Não foi possível buscar localização: %v (continuando sem localização)", i+1, len(pageLinks), err)
			} else if len(location) >= 1 {
				images[i].Location = location[0]
				log.Printf("[JetPhotos] [Imagem %d/%d] Localização: %s", i+1, len(pageLinks), location[0])
			} else {
				log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: Nenhuma localização encontrada. Encontrado: %v", i+1, len(pageLinks), location)
			}
		}

		// photographer (opcional - não retorna erro se não encontrar)
		log.Printf("[JetPhotos] [Imagem %d/%d] Buscando fotógrafo...", i+1, len(pageLinks))
		photographer, err := s.ScrapeText("h6", "header-reset", 1)
		if err != nil {
			log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: Não foi possível buscar fotógrafo: %v (continuando sem fotógrafo)", i+1, len(pageLinks), err)
		} else if len(photographer) >= 1 {
			images[i].Photographer = photographer[0]
			log.Printf("[JetPhotos] [Imagem %d/%d] Fotógrafo: %s", i+1, len(pageLinks), photographer[0])
		} else {
			log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: Nenhum fotógrafo encontrado. Encontrado: %v", i+1, len(pageLinks), photographer)
		}
		
		log.Printf("[JetPhotos] [Imagem %d/%d] Processamento concluído com sucesso", i+1, len(pageLinks))
		return nil
	}

	g, _ := errgroup.WithContext(context.Background())

	for i, link := range pageLinks {
		// Capturar variáveis para evitar problema de closure
		idx := i
		pageLink := link
		g.Go(func() error {
			err := pageScraper(idx, pageLink)
			if err != nil {
				log.Printf("[JetPhotos] [Imagem %d/%d] AVISO: Erro no processamento (mas continuando): %v", idx+1, len(pageLinks), err)
				// Não retornar erro para não interromper outras imagens
				// A imagem ainda pode ter Image e Thumbnail válidos mesmo sem metadados
			}
			return nil // Sempre retornar nil para não interromper outras imagens
		})
	}

	log.Printf("[JetPhotos] Aguardando conclusão do processamento paralelo...")
	// Não retornar erro mesmo se alguns processamentos falharem
	// (erros já foram logados individualmente)
	_ = g.Wait()
	log.Printf("[JetPhotos] Processamento paralelo concluído")

	// Verificar imagens processadas e filtrar apenas as válidas
	validImages := 0
	for i, img := range images {
		if img.Image != "" || img.Thumbnail != "" {
			validImages++
			log.Printf("[JetPhotos] Imagem %d válida - Image: %s, Thumbnail: %s", i+1, img.Image, img.Thumbnail)
		} else {
			log.Printf("[JetPhotos] AVISO: Imagem %d está vazia (Image: '%s', Thumbnail: '%s')", i+1, img.Image, img.Thumbnail)
		}
	}

	result := &JetPhotosResult{
		Images: images,
		Reg:    strings.ToUpper(reg),
	}

	log.Printf("[JetPhotos] Busca concluída para %s: %d imagens válidas de %d processadas", reg, validImages, len(images))
	return result, nil
}

func jpError(msg, reg, url string, err error) error {
	return fmt.Errorf("Error %s for %s at %s: %v", msg, reg, url, err)
}
