package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type application struct {
	errorLog      *log.Logger
	infoLog       *log.Logger
	templateCache map[string]*template.Template

	statsMu      sync.Mutex
	apiCalls     int
	totalLatency time.Duration
}

func getBaseDir() string {
	ex, err := os.Executable()
	if err != nil {
		// Fallback para diretório de trabalho atual
		wd, _ := os.Getwd()
		return wd
	}
	exPath := filepath.Dir(ex)
	// Se estiver em ./bin/jetapi, sobe um nível para a raiz do projeto
	if filepath.Base(exPath) == "bin" {
		return filepath.Dir(exPath)
	}
	return exPath
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

	baseDir := getBaseDir()
	infoLog.Printf("Base directory: %s", baseDir)

	templateCache, err := newTemplateCache(baseDir)
	if err != nil {
		errorLog.Fatal(err)
	}

	app := &application{
		errorLog:      errorLog,
		infoLog:       infoLog,
		templateCache: templateCache,
	}

	srv := &http.Server{
		Addr:     addr,
		ErrorLog: errorLog,
		Handler:  app.routes(baseDir),
	}

	app.infoLog.Print("Starting stats logger")
	go app.statsLogger()

	app.infoLog.Printf("Starting server on %s", addr)
	err = srv.ListenAndServe()
	errorLog.Fatal(err)
}
