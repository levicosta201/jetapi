package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/macsencasaus/jetapi/internal/sites"
)

func (app *application) logErr(err error) {
	trace := fmt.Sprintf("%v\n%s", err, debug.Stack())
	app.errorLog.Println(trace)
	
	// Em modo de desenvolvimento, também imprime no console de forma mais visível
	if app.devMode {
		fmt.Fprintf(os.Stderr, "\n========== ERROR ==========\n")
		fmt.Fprintf(os.Stderr, "%v\n", err)
		fmt.Fprintf(os.Stderr, "===========================\n\n")
	}
}

func (app *application) serverError(w http.ResponseWriter, err error) {
	app.logErr(err)
	status := http.StatusInternalServerError
	
	// Verificar se o header já foi escrito
	if w.Header().Get("Content-Type") != "" {
		return
	}
	
	// Em modo de desenvolvimento, mostra o erro detalhado na resposta
	if app.devMode {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(status)
		fmt.Fprintf(w, "Internal Server Error\n\nError: %v\n\nStack Trace:\n%s", err, debug.Stack())
	} else {
		http.Error(w, http.StatusText(status), status)
	}
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

func (app *application) badRequest(w http.ResponseWriter) {
	app.clientError(w, http.StatusBadRequest)
}

func (app *application) render(
	w http.ResponseWriter,
	status int,
	page string,
	data *sites.ScrapeResult,
) {
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverError(w, err)
		return
	}

	// Garantir que data não seja nil e tenha estruturas inicializadas
	if data != nil {
		if data.JetPhotos == nil {
			data.JetPhotos = &sites.JetPhotosResult{Images: []sites.ImageAttributes{}}
		}
		if data.FlightRadar == nil {
			data.FlightRadar = &sites.FlightRadarResult{Flights: []*sites.FlightAttributes{}}
		}
	} else {
		data = &sites.ScrapeResult{
			JetPhotos:   &sites.JetPhotosResult{Images: []sites.ImageAttributes{}},
			FlightRadar: &sites.FlightRadarResult{Flights: []*sites.FlightAttributes{}},
		}
	}

	// Verificar se o header já foi escrito antes de escrever
	if w.Header().Get("Content-Type") == "" {
		w.WriteHeader(status)
	}

	err := ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		// Não chamar serverError aqui para evitar WriteHeader duplo
		// Apenas logar o erro
		app.logErr(fmt.Errorf("template execution error: %v", err))
		if app.devMode {
			// Se ainda não escreveu nada, mostrar erro
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "Template execution error: %v", err)
			}
		}
	}
}

func (app *application) parseAPIQueries(
	w http.ResponseWriter,
	r *http.Request,
) (*sites.APIQueries, error) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		return nil, fmt.Errorf("%d", http.StatusMethodNotAllowed)
	}
	queryParams := r.URL.Query()

	reg := queryParams.Get("reg")
	if reg == "" {
		return nil, fmt.Errorf("%d", http.StatusNotFound)
	}

	// is alphanumeric and may include '-'
	isAlpha := regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString(reg)
	if !isAlpha {
		return nil, fmt.Errorf("%d", http.StatusBadRequest)
	}

	onlyJP := queryParams.Get("only_jp") == "true"
	onlyFR := queryParams.Get("only_fr") == "true"

	photos, err := handleNumQuery(queryParams, "photos")
	if err != nil {
		return nil, err
	}
	if photos == -1 {
		photos = 3
	}

	flights, err := handleNumQuery(queryParams, "flights")
	if err != nil {
		return nil, err
	}
	if flights == -1 {
		flights = 20
	}

	q := &sites.APIQueries{
		Reg:     reg,
		Photos:  photos,
		Flights: flights,
		OnlyJP:  onlyJP,
		OnlyFR:  onlyFR,
	}
	return q, nil
}

func handleNumQuery(qp url.Values, query string) (int, error) {
	resStr := qp.Get(query)
	if resStr == "" {
		return -1, nil
	}
	res, err := strconv.Atoi(resStr)
	if err != nil || res < 0 {
		return 0, fmt.Errorf("%d", http.StatusBadRequest)
	}
	return res, nil
}

func newTemplateCache(baseDir string) (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pagesPath := filepath.Join(baseDir, "ui", "html", "pages", "*.tmpl.html")
	pages, err := filepath.Glob(pagesPath)
	if err != nil {
		return nil, fmt.Errorf("error globbing pages: %v", err)
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no template pages found in %s", pagesPath)
	}

	basePath := filepath.Join(baseDir, "ui", "html", "base.tmpl.html")
	partialsPath := filepath.Join(baseDir, "ui", "html", "partial", "*.tmpl.html")

	for _, page := range pages {
		name := filepath.Base(page)

		ts, err := template.ParseFiles(basePath)
		if err != nil {
			return nil, fmt.Errorf("error parsing base template: %v", err)
		}

		ts, err = ts.ParseGlob(partialsPath)
		if err != nil {
			return nil, fmt.Errorf("error parsing partials: %v", err)
		}

		ts, err = ts.ParseFiles(page)
		if err != nil {
			return nil, fmt.Errorf("error parsing page %s: %v", page, err)
		}

		cache[name] = ts
	}

	return cache, nil
}

func (app *application) statsLogger() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		app.statsMu.Lock()

		var avgLatency time.Duration
		if app.apiCalls > 0 {
			avgLatency = app.totalLatency / time.Duration(app.apiCalls)
		}

		msg := `
+-----------------------------+
|        API Statistics       |
+-----------------------------+
| Calls to /api     : %7d |
| Average Latency   : %7s |
+-----------------------------+`

		app.infoLog.Printf(msg, app.apiCalls, avgLatency.Round(time.Millisecond))

		app.apiCalls = 0
		app.totalLatency = 0

		app.statsMu.Unlock()
	}
}
