package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/macsencasaus/jetapi/internal/cache"
	"github.com/macsencasaus/jetapi/internal/cloudfront"
	"github.com/macsencasaus/jetapi/internal/s3"
	"github.com/macsencasaus/jetapi/internal/sites"
)

type application struct {
	errorLog      *log.Logger
	infoLog       *log.Logger
	templateCache map[string]*template.Template
	devMode       bool

	statsMu      sync.Mutex
	apiCalls     int
	totalLatency time.Duration
}

func main() {
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	// Inicializar cache (Redis)
	var cacheInstance *cache.Cache
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			redisDB = db
		}
	}

	if redisHost != "" || redisPort != "" {
		cacheConfig := &cache.CacheConfig{
			Host:     redisHost,
			Port:     redisPort,
			Password: redisPassword,
			DB:       redisDB,
			TTL:      24 * time.Hour,
		}
		c, err := cache.NewCache(cacheConfig)
		if err != nil {
			errorLog.Printf("Warning: Failed to connect to Redis cache: %v. Continuing without cache.", err)
		} else {
			cacheInstance = c
			infoLog.Print("Redis cache connected successfully")
		}
	}

	// Inicializar CloudFront
	var cloudFrontInstance *cloudfront.CloudFront
	cfDomain := os.Getenv("CLOUDFRONT_DOMAIN")
	cfKeyPairID := os.Getenv("CLOUDFRONT_KEY_PAIR_ID")
	cfPrivateKey := os.Getenv("CLOUDFRONT_PRIVATE_KEY")
	cfTTL := int64(3600)
	if ttlStr := os.Getenv("CLOUDFRONT_TTL"); ttlStr != "" {
		if ttl, err := strconv.ParseInt(ttlStr, 10, 64); err == nil {
			cfTTL = ttl
		}
	}

	if cfDomain != "" {
		cfConfig := &cloudfront.CloudFrontConfig{
			DomainName: cfDomain,
			KeyPairID:  cfKeyPairID,
			PrivateKey: cfPrivateKey,
			TTL:        cfTTL,
		}
		cf, err := cloudfront.NewCloudFront(cfConfig)
		if err != nil {
			errorLog.Printf("Warning: Failed to initialize CloudFront: %v. Continuing without CloudFront.", err)
		} else {
			cloudFrontInstance = cf
			infoLog.Print("CloudFront configured successfully")
		}
	}

	// Inicializar S3
	var s3ManagerInstance *s3.S3Manager
	s3Region := os.Getenv("AWS_REGION")
	s3Bucket := os.Getenv("S3_BUCKET")
	s3AccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	s3SecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	s3BasePath := os.Getenv("S3_BASE_PATH")
	if s3BasePath == "" {
		s3BasePath = "jetapi"
	}

	if s3Bucket != "" && s3AccessKey != "" && s3SecretKey != "" {
		s3Config := &s3.S3Config{
			Region:          s3Region,
			Bucket:          s3Bucket,
			AccessKeyID:     s3AccessKey,
			SecretAccessKey: s3SecretKey,
			BasePath:        s3BasePath,
		}
		s3Mgr, err := s3.NewS3Manager(s3Config)
		if err != nil {
			errorLog.Printf("Warning: Failed to initialize S3: %v. Continuing without S3.", err)
		} else {
			s3ManagerInstance = s3Mgr
			infoLog.Print("S3 configured successfully")
		}
	}

	// Configurar cache manager, CloudFront e S3 nos sites
	cacheMgr := cache.NewManager(cacheInstance, cloudFrontInstance, s3ManagerInstance)
	sites.SetCacheManager(cacheMgr)
	if cloudFrontInstance != nil {
		sites.SetCloudFront(cloudFrontInstance)
	}
	// O cacheManager já está configurado através de SetCacheManager, não precisa de função separada

	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	// Verificar se está em modo de desenvolvimento
	devMode := os.Getenv("DEV_MODE") == "true" || os.Getenv("ENV") == "development"

	app := &application{
		errorLog:      errorLog,
		infoLog:       infoLog,
		templateCache: templateCache,
		devMode:       devMode,
	}

	if devMode {
		infoLog.Print("Running in DEVELOPMENT mode - detailed errors will be shown")
	}

	srv := &http.Server{
		Addr:     addr,
		ErrorLog: errorLog,
		Handler:  app.routes(),
	}

	app.infoLog.Print("Starting stats logger")
	go app.statsLogger()

	app.infoLog.Printf("Starting server on %s", addr)
	err = srv.ListenAndServe()
	errorLog.Fatal(err)
}
