package s3

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	BasePath        string // Caminho base dentro do bucket (ex: "jetapi")
}

type S3Manager struct {
	config  *S3Config
	session *session.Session
	client  *s3.S3
}

func NewS3Manager(config *S3Config) (*S3Manager, error) {
	if config == nil {
		return nil, fmt.Errorf("s3 config cannot be nil")
	}

	if config.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket name is required")
	}

	if config.Region == "" {
		config.Region = "us-east-1" // Padrão
	}

	if config.BasePath == "" {
		config.BasePath = "jetapi"
	}

	// Configurar credenciais AWS
	creds := credentials.NewStaticCredentials(
		config.AccessKeyID,
		config.SecretAccessKey,
		"",
	)

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(config.Region),
		Credentials: creds,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}

	client := s3.New(sess)

	return &S3Manager{
		config:  config,
		session: sess,
		client:  client,
	}, nil
}

// UploadImage baixa uma imagem de uma URL e faz upload para o S3
// Retorna a chave (path) do objeto no S3
func (m *S3Manager) UploadImage(imageURL string) (string, error) {
	// Baixar a imagem
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
	}

	// Ler o conteúdo da imagem
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %v", err)
	}

	// Gerar nome do arquivo baseado na URL
	fileName := generateFileName(imageURL)

	// Caminho completo no S3
	s3Key := fmt.Sprintf("%s/%s", m.config.BasePath, fileName)

	// Fazer upload para S3
	_, err = m.client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(m.config.Bucket),
		Key:           aws.String(s3Key),
		Body:          bytes.NewReader(imageData),
		ContentType:   aws.String(getContentType(imageURL)),
		ContentLength: aws.Int64(int64(len(imageData))),
		CacheControl: aws.String("max-age=31536000"), // Cache por 1 ano
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %v", err)
	}

	return s3Key, nil
}

// UploadImageFromBytes faz upload de uma imagem a partir de bytes
func (m *S3Manager) UploadImageFromBytes(data []byte, fileName string) (string, error) {
	// Caminho completo no S3
	s3Key := fmt.Sprintf("%s/%s", m.config.BasePath, fileName)

	// Fazer upload para S3
	_, err := m.client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(m.config.Bucket),
		Key:           aws.String(s3Key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String("image/jpeg"), // Padrão
		ContentLength: aws.Int64(int64(len(data))),
		CacheControl:  aws.String("max-age=31536000"),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %v", err)
	}

	return s3Key, nil
}

// CheckIfExists verifica se uma imagem já existe no S3
func (m *S3Manager) CheckIfExists(s3Key string) (bool, error) {
	_, err := m.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(m.config.Bucket),
		Key:    aws.String(s3Key),
	})

	if err != nil {
		// Se o erro for "NotFound", a imagem não existe
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetS3Key gera a chave S3 para uma URL de imagem
func (m *S3Manager) GetS3Key(imageURL string) string {
	fileName := generateFileName(imageURL)
	return fmt.Sprintf("%s/%s", m.config.BasePath, fileName)
}

// generateFileName gera um nome de arquivo único baseado na URL
func generateFileName(url string) string {
	// Extrair o nome do arquivo da URL
	baseName := filepath.Base(url)
	
	// Se não tiver extensão, adicionar .jpg como padrão
	if !strings.Contains(baseName, ".") {
		baseName += ".jpg"
	}

	// Adicionar hash da URL para garantir unicidade
	hash := simpleHash(url)
	ext := filepath.Ext(baseName)
	name := strings.TrimSuffix(baseName, ext)
	
	// Combinar: nome_hash.ext
	return fmt.Sprintf("%s_%s%s", name, hash, ext)
}

// simpleHash gera um hash simples da URL
func simpleHash(s string) string {
	h := 0
	for _, char := range s {
		h = h*31 + int(char)
		h = h & 0x7FFFFFFF // Manter positivo
	}
	return fmt.Sprintf("%08x", h)
}

// getContentType determina o content type baseado na URL
func getContentType(url string) string {
	ext := strings.ToLower(filepath.Ext(url))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// IsEnabled verifica se o S3 está configurado
func (m *S3Manager) IsEnabled() bool {
	return m != nil && m.config != nil && m.config.Bucket != ""
}

