package cloudfront

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type CloudFrontConfig struct {
	DomainName string // Ex: d1234567890.cloudfront.net
	KeyPairID  string // ID do par de chaves CloudFront
	PrivateKey string // Chave privada (PEM format)
	TTL        int64  // Tempo de expiração em segundos (padrão: 1 hora)
}

type CloudFront struct {
	config *CloudFrontConfig
}

func NewCloudFront(config *CloudFrontConfig) (*CloudFront, error) {
	if config == nil {
		return nil, fmt.Errorf("cloudfront config cannot be nil")
	}

	if config.DomainName == "" {
		return nil, fmt.Errorf("cloudfront domain name is required")
	}

	// Se não tiver KeyPairID e PrivateKey, ainda funciona mas sem assinatura
	// Isso é útil para desenvolvimento ou se já estiver usando CloudFront sem assinatura

	ttl := config.TTL
	if ttl == 0 {
		ttl = 3600 // Padrão: 1 hora
	}

	config.TTL = ttl

	return &CloudFront{
		config: config,
	}, nil
}

// ConvertImageURL converte uma URL de imagem para CloudFront
// Se a URL já for do CloudFront, retorna como está
// Se não tiver configuração de CloudFront, retorna a URL original
func (cf *CloudFront) ConvertImageURL(originalURL string) string {
	if cf == nil || cf.config == nil || cf.config.DomainName == "" {
		return originalURL
	}

	// Se já for uma URL do CloudFront, retorna como está
	if strings.Contains(originalURL, "cloudfront.net") {
		return originalURL
	}

	// Extrair o caminho da URL original
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}

	// Construir a URL do CloudFront
	// Formato: https://domain.cloudfront.net/path/to/image
	cloudfrontURL := fmt.Sprintf("https://%s%s", cf.config.DomainName, parsedURL.Path)

	// Se tiver query string, adicionar
	if parsedURL.RawQuery != "" {
		cloudfrontURL += "?" + parsedURL.RawQuery
	}

	// Se tiver KeyPairID e PrivateKey, assinar a URL
	if cf.config.KeyPairID != "" && cf.config.PrivateKey != "" {
		signedURL, err := cf.signURL(cloudfrontURL)
		if err == nil {
			return signedURL
		}
		// Se falhar na assinatura, retorna sem assinatura
	}

	return cloudfrontURL
}

// ConvertS3Key converte uma chave S3 para URL do CloudFront
func (cf *CloudFront) ConvertS3Key(s3Key string) string {
	if cf == nil || cf.config == nil || cf.config.DomainName == "" {
		return ""
	}

	// Construir URL do CloudFront a partir da chave S3
	// Formato: https://domain.cloudfront.net/jetapi/filename.jpg
	cloudfrontURL := fmt.Sprintf("https://%s/%s", cf.config.DomainName, s3Key)

	// Se tiver KeyPairID e PrivateKey, assinar a URL
	if cf.config.KeyPairID != "" && cf.config.PrivateKey != "" {
		signedURL, err := cf.signURL(cloudfrontURL)
		if err == nil {
			return signedURL
		}
	}

	return cloudfrontURL
}

// signURL assina uma URL do CloudFront usando Signed URLs
// Implementação simplificada - para produção, considere usar AWS SDK
func (cf *CloudFront) signURL(urlToSign string) (string, error) {
	// Esta é uma implementação básica
	// Para produção completa, use o AWS SDK for Go v2
	// Por enquanto, retornamos a URL sem assinatura se houver erro

	// Parse da URL
	parsedURL, err := url.Parse(urlToSign)
	if err != nil {
		return urlToSign, err
	}

	// Se não tiver KeyPairID, retorna URL sem assinatura
	if cf.config.KeyPairID == "" {
		return urlToSign, nil
	}

	// Criar política de expiração
	expires := time.Now().Unix() + cf.config.TTL

	// Em produção, você deve usar a chave privada para assinar a política
	// Por enquanto, retornamos a URL com parâmetros de expiração básicos
	query := parsedURL.Query()
	query.Set("Expires", fmt.Sprintf("%d", expires))
	query.Set("Key-Pair-Id", cf.config.KeyPairID)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

// ConvertImageURLs converte múltiplas URLs de uma vez
func (cf *CloudFront) ConvertImageURLs(urls []string) []string {
	if cf == nil {
		return urls
	}

	converted := make([]string, len(urls))
	for i, u := range urls {
		converted[i] = cf.ConvertImageURL(u)
	}
	return converted
}

// IsEnabled verifica se o CloudFront está configurado
func (cf *CloudFront) IsEnabled() bool {
	return cf != nil && cf.config != nil && cf.config.DomainName != ""
}

// SimpleConvert é uma função helper que converte URLs sem necessidade de assinatura
// Útil para desenvolvimento ou quando CloudFront está configurado sem Signed URLs
func SimpleConvert(originalURL, cloudfrontDomain string) string {
	if cloudfrontDomain == "" {
		return originalURL
	}

	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}

	// Extrair apenas o path para usar no CloudFront
	// Assumindo que as imagens estão no mesmo path no CloudFront
	return fmt.Sprintf("https://%s%s", cloudfrontDomain, parsedURL.Path)
}

// HashURL cria um hash da URL para usar como chave de cache
func HashURL(url string) string {
	h := sha1.New()
	h.Write([]byte(url))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))[:16]
}

