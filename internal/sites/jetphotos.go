package sites

import (
	"fmt"
	"strings"

	"context"
	"github.com/macsencasaus/jetapi/internal/cache"
	"github.com/macsencasaus/jetapi/internal/cloudfront"
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

var cloudFrontConverter *cloudfront.CloudFront

// SetCloudFront configura o conversor CloudFront para URLs de imagens
func SetCloudFront(cf *cloudfront.CloudFront) {
	cloudFrontConverter = cf
}

// getCacheManager retorna o cache manager global (definido em sites.go)
func getCacheManager() *cache.Manager {
	return cacheManager
}

func ScrapeJetPhotos(q *APIQueries) (*JetPhotosResult, error) {
	reg := q.Reg
	if q.Photos == 0 {
		return &JetPhotosResult{Reg: strings.ToUpper(reg)}, nil
	}

	URL := fmt.Sprintf("%s/photo/keyword/%s", jpHomeURL, reg)
	b, err := scraper.FetchHTML(URL)
	if err != nil {
		return nil, jpError("scraping search URL", reg, URL, err)
	}

	s := scraper.NewScraper(b)
	defer s.Close()

	pageLinks := []string{}
	thumbnails := []string{}
	for i := 0; i < q.Photos; i++ {
		pageLink, err := s.ScrapeLinks("a", "result__photoLink", 1)
		if err != nil {
			if len(pageLinks) > 0 {
				break
			}
			return nil, jpError("scraping aircraft pagelinks", reg, URL, err)
		}

		thumbnail, err := s.ScrapeLinks("img", "result__photo", 1)
		if err != nil {
			if len(thumbnails) > 0 {
				break
			}
			return nil, jpError("scraping aircraft thumbnails", reg, URL, err)
		}
		pageLinks = append(pageLinks, pageLink[0])
		thumbnails = append(thumbnails, thumbnail[0])
	}

	images := make([]ImageAttributes, len(pageLinks))

	pageScraper := func(i int, link string) error {
		photoURL := fmt.Sprintf("%s%s", jpHomeURL, link)
		images[i].Link = photoURL
		originalThumbnailURL := "https:" + thumbnails[i]
		
		// Processar thumbnail: baixar, fazer upload para S3 e gerar URL CloudFront
		finalThumbnailURL := processImage(originalThumbnailURL)
		images[i].Thumbnail = finalThumbnailURL

		b, err := scraper.FetchHTML(photoURL)
		if err != nil {
			return jpError("fetching HTML page", reg, URL, err)
		}

		s := scraper.NewScraper(b)
		defer s.Close()

		// photo links
		photoLinkArr, err := s.ScrapeLinks("img", "large-photo__img", 1)
		if err != nil {
			return jpError("scraping photo links", reg, URL, err)
		}
		originalImageURL := photoLinkArr[0]
		
		// Processar imagem: baixar, fazer upload para S3 e gerar URL CloudFront
		finalImageURL := processImage(originalImageURL)
		images[i].Image = finalImageURL

		// registration + dates
		res, err := s.ScrapeText("h4", "headerText4 color-shark", 3)
		if err != nil {
			return jpError("scraping registrating text", reg, URL, err)
		}
		images[i].DateTaken = res[1]
		images[i].DateUploaded = res[2]

		// aircraft
		s.Advance("h2", "header-reset", 1)
		res, err = s.ScrapeText("a", "link", 3)
		if err != nil {
			return jpError("scraping aircraft text", reg, URL, err)
		}
		images[i].Aircraft = res[0]
		images[i].Airline = res[1]
		images[i].Serial = strings.TrimSpace(res[2])

		// location
		s.Advance("h5", "header-reset", 1)
		location, err := s.ScrapeText("a", "link", 1)
		if err != nil {
			return jpError("scraping location text", reg, URL, err)
		}
		images[i].Location = location[0]

		// photographer
		photographer, err := s.ScrapeText("h6", "header-reset", 1)
		if err != nil {
			return jpError("scraping photographer text", reg, URL, err)
		}
		images[i].Photographer = photographer[0]

		return nil
	}

	g, _ := errgroup.WithContext(context.Background())

	for i, link := range pageLinks {
		g.Go(func() error {
			return pageScraper(i, link)
		})
	}

	if err = g.Wait(); err != nil {
		return nil, err
	}

	result := &JetPhotosResult{
		Images: images,
		Reg:    strings.ToUpper(reg),
	}

	return result, nil
}

// processImage processa uma imagem: verifica cache, baixa, faz upload para S3 e retorna URL CloudFront
func processImage(imageURL string) string {
	cm := getCacheManager()
	if cm == nil {
		// Se não tiver S3/CloudFront configurado, retornar URL original
		if cloudFrontConverter != nil && cloudFrontConverter.IsEnabled() {
			return cloudFrontConverter.ConvertImageURL(imageURL)
		}
		return imageURL
	}

	s3Manager := cm.GetS3Manager()
	cf := cm.GetCloudFront()

	// Se S3 não estiver configurado, apenas converter para CloudFront se disponível
	if s3Manager == nil || !s3Manager.IsEnabled() {
		if cf != nil && cf.IsEnabled() {
			return cf.ConvertImageURL(imageURL)
		}
		return imageURL
	}

	// Gerar chave S3
	s3Key := s3Manager.GetS3Key(imageURL)

	// Verificar se já existe no S3 (cache)
	exists, err := s3Manager.CheckIfExists(s3Key)
	if err == nil && exists {
		// Imagem já existe no S3, gerar URL CloudFront
		if cf != nil && cf.IsEnabled() {
			return cf.ConvertS3Key(s3Key)
		}
		// Se não tiver CloudFront, retornar URL original
		return imageURL
	}

	// Imagem não existe, fazer upload para S3
	uploadedKey, err := s3Manager.UploadImage(imageURL)
	if err != nil {
		// Se falhar o upload, retornar URL original
		if cf != nil && cf.IsEnabled() {
			return cf.ConvertImageURL(imageURL)
		}
		return imageURL
	}

	// Gerar URL CloudFront para a imagem no S3
	if cf != nil && cf.IsEnabled() {
		return cf.ConvertS3Key(uploadedKey)
	}

	// Se não tiver CloudFront, retornar URL original
	return imageURL
}

func jpError(msg, reg, url string, err error) error {
	return fmt.Errorf("Error %s for %s at %s: %v", msg, reg, url, err)
}
