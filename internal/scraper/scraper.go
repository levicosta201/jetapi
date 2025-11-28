package scraper

import (
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"golang.org/x/net/html"
)

type Scraper struct {
	body      io.ReadCloser
	tokenizer *html.Tokenizer
	tokens    []html.Token
}

type ActionType uint32

const (
	SCRAPE ActionType = iota
	ADVANCE
)

func NewScraper(body io.ReadCloser) *Scraper {
	return &Scraper{body: body}
}

func (s *Scraper) Close() {
	s.body.Close()
}

func (s *Scraper) ScrapeLinks(startTag, class string, count int) ([]string, error) {
	log.Printf("[Scraper] ScrapeLinks: buscando tag '%s' com classe '%s', count=%d", startTag, class, count)
	tokens, err := s.scrapeNextTokens(startTag, class, count, SCRAPE, html.StartTagToken)
	if err != nil {
		log.Printf("[Scraper] ScrapeLinks: ERRO ao buscar tokens - %v", err)
		return nil, err
	}
	log.Printf("[Scraper] ScrapeLinks: encontrados %d tokens", len(tokens))
	links := make([]string, len(tokens))
	for i, tk := range tokens {
		for _, attr := range tk.Attr {
			if attr.Key == "href" || attr.Key == "src" || attr.Key == "srcset" {
				links[i] = attr.Val
				log.Printf("[Scraper] ScrapeLinks: link %d encontrado: %s (atributo: %s)", i+1, attr.Val, attr.Key)
				break
			}
		}
		if links[i] == "" {
			log.Printf("[Scraper] ScrapeLinks: AVISO - token %d não tem href/src/srcset", i+1)
		}
	}
	log.Printf("[Scraper] ScrapeLinks: retornando %d links", len(links))
	return links, nil
}

func (s *Scraper) ScrapeText(startTag, class string, count int) ([]string, error) {
	log.Printf("[Scraper] ScrapeText: buscando tag '%s' com classe '%s', count=%d", startTag, class, count)
	tokens, err := s.scrapeNextTokens(startTag, class, count, SCRAPE, html.TextToken)
	if err != nil {
		log.Printf("[Scraper] ScrapeText: ERRO ao buscar tokens - %v", err)
		return nil, err
	}
	log.Printf("[Scraper] ScrapeText: encontrados %d tokens", len(tokens))
	data := make([]string, len(tokens))
	for i := 0; i < len(data); i++ {
		data[i] = tokens[i].Data
		log.Printf("[Scraper] ScrapeText: texto %d: '%s'", i+1, strings.TrimSpace(data[i]))
	}
	return data, nil
}

func (s *Scraper) Advance(startTag, class string, count int) error {
	_, err := s.scrapeNextTokens(startTag, class, count, ADVANCE, html.StartTagToken)
	return err
}

func (s *Scraper) TryScrapeText() (string, bool) {
	tt := s.tokenizer.Next()
	t := s.tokenizer.Token()

	for strings.TrimSpace(t.Data) == "" {
		tt = s.tokenizer.Next()
		t = s.tokenizer.Token()
	}

	if tt != html.TextToken {
		s.tokens = append(s.tokens, t)
		return "", false
	}

	return t.Data, true
}

func (s *Scraper) scrapeNextTokens(
	startTag, class string,
	count int,
	action ActionType,
	tt html.TokenType,
) ([]html.Token, error) {
	if s.tokenizer == nil {
		s.tokenizer = html.NewTokenizer(s.body)
	}
	var resultTokens []html.Token
	atLeastOne := false

	for count > 0 {
		token, err := s.nextToken(html.StartTagToken, startTag, class)

		if err != nil {
			if atLeastOne {
				break
			}
			return nil, err
		}

		if tt == html.TextToken {
			s.tokenizer.Next()
			token = s.tokenizer.Token()
		}

		if action == SCRAPE {
			resultTokens = append(resultTokens, token)
			atLeastOne = true
		}

		count--
	}

	return resultTokens, nil
}

func (s *Scraper) nextToken(tt html.TokenType, data, class string) (html.Token, error) {
	log.Printf("[Scraper] nextToken: buscando token tipo=%d, tag='%s', class='%s'", tt, data, class)
	
	// Primeiro verificar tokens já em cache
	for i, t := range s.tokens {
		if t.Type == tt && t.Data == data && tokenHasClass(&t, class) {
			log.Printf("[Scraper] nextToken: encontrado em cache (índice %d)", i)
			s.tokens = s.tokens[i+1:]
			return t, nil
		}
	}

	var nilToken html.Token
	tokenCount := 0
	for {
		tokenType := s.tokenizer.Next()
		tokenCount++
		if tokenCount%1000 == 0 {
			log.Printf("[Scraper] nextToken: processados %d tokens, ainda procurando...", tokenCount)
		}
		
		if tokenType == html.ErrorToken {
			if s.tokenizer.Err() == io.EOF {
				log.Printf("[Scraper] nextToken: EOF alcançado após %d tokens, tag '%s' com classe '%s' não encontrada", tokenCount, data, class)
				return nilToken, s.Errorf("tag '%s' with class '%s' not found", data, class)
			}
			return nilToken, s.Errorf("Error tokenizing html: %v", s.tokenizer.Err())
		}
		t := s.tokenizer.Token()
		s.tokens = append(s.tokens, t)

		if tokenType == tt && t.Data == data && tokenHasClass(&t, class) {
			log.Printf("[Scraper] nextToken: encontrado após %d tokens", tokenCount)
			s.tokens = s.tokens[len(s.tokens):]
			return t, nil
		}
	}
}

func (s *Scraper) Errorf(format string, a ...any) error {
	return fmt.Errorf("Scraper Error: %s", fmt.Sprintf(format, a...))
}

// tokenHasClass verifica se um token HTML tem a classe especificada.
// Suporta correspondência parcial de classes (ex: "result__photoLink" irá corresponder a "result__photoLink active").
func tokenHasClass(t *html.Token, class string) bool {
	if class == "" {
		return true
	}

	for _, attr := range t.Attr {
		if attr.Key == "class" {
			// Dividir a string de classes em classes individuais
			classes := strings.Fields(attr.Val)
			for _, c := range classes {
				if c == class {
					return true
				}
			}
			// Log apenas quando não encontrou (para debug)
			// log.Printf("[Scraper] tokenHasClass: tag '%s' tem classes '%s', procurando '%s' - NÃO encontrado", t.Data, attr.Val, class)
		}
	}
	return false
}

func FetchHTML(URL string) (io.ReadCloser, error) {
	log.Printf("[Scraper] FetchHTML: iniciando requisição para %s", URL)
	
	tlsConfig := &tls.Config{
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
		MinVersion:       tls.VersionTLS12,
		MaxVersion:       tls.VersionTLS13,
		CurvePreferences: []tls.CurveID{tls.CurveP256, tls.X25519},
	}

	// Obter gerenciador de proxies
	proxyManager := GetProxyManager()
	
	// Rotacionar proxy para esta nova requisição (cada API call usa proxy diferente)
	var initialProxyURL string
	if proxyManager.HasProxies() {
		initialProxyURL = proxyManager.GetNextProxy() // Rotaciona para nova requisição
		log.Printf("[Scraper] FetchHTML: nova requisição - usando proxy (rotação automática)")
	} else {
		initialProxyURL = ""
		log.Printf("[Scraper] FetchHTML: nova requisição - tentando sem proxy (nenhum proxy configurado)")
	}

	var lastErr error
	maxRetries := 1
	if proxyManager.HasProxies() {
		// Se houver proxies, tentar todos eles se um falhar
		maxRetries = len(proxyManager.GetAllProxies()) + 1 // +1 para tentar sem proxy no final
	}

	proxyURL := initialProxyURL

	for retry := 0; retry < maxRetries; retry++ {
		if proxyURL != "" {
			log.Printf("[Scraper] FetchHTML: tentativa com proxy %d/%d", retry+1, maxRetries)
		} else {
			log.Printf("[Scraper] FetchHTML: tentativa sem proxy %d/%d", retry+1, maxRetries)
		}

		// Criar transport com ou sem proxy
		transport := &http.Transport{
			TLSClientConfig: tlsConfig,
		}

		if proxyURL != "" {
			proxyURLParsed, err := url.Parse(proxyURL)
			if err != nil {
				log.Printf("[Scraper] FetchHTML: ERRO ao parsear proxy URL: %v", err)
				// Tentar próximo proxy ou sem proxy
				if proxyManager.HasProxies() && retry < maxRetries-1 {
					proxyURL = proxyManager.GetNextProxy() // Rotaciona para próxima tentativa
					continue
				} else {
					proxyURL = "" // Tentar sem proxy
					continue
				}
			}
			transport.Proxy = http.ProxyURL(proxyURLParsed)
		}

		// Criar cookie jar para manter sessão/cookies
		jar, err := cookiejar.New(nil)
		if err != nil {
			log.Printf("[Scraper] FetchHTML: AVISO - não foi possível criar cookie jar: %v", err)
			jar = nil
		}

		client := &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
			Jar:       jar,
		}

		req, err := http.NewRequest("GET", URL, nil)
		if err != nil {
			log.Printf("[Scraper] FetchHTML: ERRO ao criar requisição - %v", err)
			lastErr = err
			transport.CloseIdleConnections()
			if proxyManager.HasProxies() && retry < maxRetries-1 {
				proxyURL = proxyManager.GetNextProxy() // Rotaciona para próxima tentativa
				continue
			} else {
				proxyURL = "" // Tentar sem proxy
				continue
			}
		}

		// Headers mais completos para evitar bloqueio (simular navegador real)
		// User-Agent variado para parecer mais real
		userAgents := []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		}
		userAgent := userAgents[retry%len(userAgents)]
		req.Header.Set("User-Agent", userAgent)
		
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9,pt-BR;q=0.8,pt;q=0.7")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		
		// Referer variado baseado na URL
		if strings.Contains(URL, "jetphotos.com") {
			req.Header.Set("Referer", "https://www.jetphotos.com/")
		} else if strings.Contains(URL, "flightradar24.com") {
			req.Header.Set("Referer", "https://www.flightradar24.com/")
		} else {
			req.Header.Set("Referer", "https://www.google.com/")
		}
		
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Cache-Control", "max-age=0")
		req.Header.Set("DNT", "1") // Do Not Track
		req.Header.Set("sec-ch-ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)

		// Fazer uma requisição inicial à página principal para obter cookies (se for JetPhotos)
		if strings.Contains(URL, "jetphotos.com") && strings.Contains(URL, "photo/keyword") {
			// Fazer requisição inicial à home para obter cookies antes de buscar
			homeURL := "https://www.jetphotos.com/"
			log.Printf("[Scraper] FetchHTML: fazendo requisição inicial à home para obter cookies")
			homeReq, _ := http.NewRequest("GET", homeURL, nil)
			homeReq.Header.Set("User-Agent", userAgent)
			homeReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
			homeReq.Header.Set("Accept-Language", "en-US,en;q=0.9")
			homeReq.Header.Set("Referer", "https://www.google.com/")
			homeResp, err := client.Do(homeReq)
			if err == nil && homeResp != nil {
				if homeResp.Body != nil {
					homeResp.Body.Close()
				}
				log.Printf("[Scraper] FetchHTML: cookies obtidos da home (status: %d)", homeResp.StatusCode)
				time.Sleep(1 * time.Second) // Pequeno delay após obter cookies
			}
		}

		var resp *http.Response
		var success bool

		// retry 3 times com delay progressivo e rotação de proxy a cada tentativa
		for i := 0; i < 3; i++ {
			// Rotacionar proxy a cada tentativa se houver proxies disponíveis
			if i > 0 && proxyManager.HasProxies() {
				proxyURL = proxyManager.GetNextProxy() // Rotaciona proxy a cada tentativa
				log.Printf("[Scraper] FetchHTML: tentativa %d/3 - rotacionando para próximo proxy", i+1)
				
				// Recriar transport com novo proxy
				transport.CloseIdleConnections()
				transport = &http.Transport{
					TLSClientConfig: tlsConfig,
				}
				if proxyURL != "" {
					proxyURLParsed, err := url.Parse(proxyURL)
					if err == nil {
						transport.Proxy = http.ProxyURL(proxyURLParsed)
					}
				}
				client.Transport = transport
				
				// Atualizar User-Agent para esta tentativa
				userAgent := userAgents[i%len(userAgents)]
				req.Header.Set("User-Agent", userAgent)
			}

			// Delay mais inteligente: aumentar progressivamente e adicionar jitter
			if i > 0 {
				baseDelay := time.Duration(i*3) * time.Second // Delay progressivo: 3s, 6s
				// Adicionar jitter aleatório entre 0.5s e 1.5s
				jitter := time.Duration(500+retry*100) * time.Millisecond
				delay := baseDelay + jitter
				log.Printf("[Scraper] FetchHTML: tentativa %d/3 após delay de %v", i+1, delay)
				time.Sleep(delay)
			} else {
				// Delay inicial variável baseado no índice do proxy
				initialDelay := time.Duration(1+retry%3) * time.Second // 1s, 2s ou 3s
				log.Printf("[Scraper] FetchHTML: tentativa %d/3 (aguardando %v antes da primeira requisição)", i+1, initialDelay)
				time.Sleep(initialDelay)
			}
			
			resp, err = client.Do(req)
			if err != nil {
				log.Printf("[Scraper] FetchHTML: ERRO ao enviar requisição (tentativa %d/3) - %v", i+1, err)
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
				if i == 2 {
					lastErr = err
					break
				}
				continue
			}

			log.Printf("[Scraper] FetchHTML: resposta recebida - Status: %d, Content-Type: %s", resp.StatusCode, resp.Header.Get("Content-Type"))

			if resp.StatusCode == http.StatusOK {
				log.Printf("[Scraper] FetchHTML: sucesso na tentativa %d/3", i+1)
				success = true
				break
			} else if i < 2 && resp.StatusCode == http.StatusForbidden {
				log.Printf("[Scraper] FetchHTML: 403 Forbidden na tentativa %d/3, tentando novamente com próximo proxy...", i+1)
				if resp.Body != nil {
					resp.Body.Close()
				}
				continue
			} else {
				log.Printf("[Scraper] FetchHTML: ERRO - código de status %d na tentativa %d/3", resp.StatusCode, i+1)
				if resp.Body != nil {
					resp.Body.Close()
				}
				lastErr = fmt.Errorf("response error code: %d, URL: %s", resp.StatusCode, URL)
				if i == 2 {
					break
				}
			}
		}

		// Se teve sucesso com este proxy, retornar
		if success {
			ctype := resp.Header.Get("Content-Type")
			log.Printf("[Scraper] FetchHTML: Content-Type final: %s", ctype)
			if !strings.HasPrefix(ctype, "text/html") {
				log.Printf("[Scraper] FetchHTML: ERRO - Content-Type não é text/html")
				if resp.Body != nil {
					resp.Body.Close()
				}
				// Fechar conexões antes de tentar próximo proxy
				transport.CloseIdleConnections()
				lastErr = fmt.Errorf("content not type text/html")
				// Tentar próximo proxy ou sem proxy
				if proxyManager.HasProxies() && retry < maxRetries-1 {
					proxyURL = proxyManager.GetNextProxy() // Rotaciona para próxima tentativa
					continue
				} else {
					proxyURL = "" // Tentar sem proxy
					continue
				}
			}

			// Verificar Content-Encoding e descomprimir se necessário
			contentEncoding := resp.Header.Get("Content-Encoding")
			log.Printf("[Scraper] FetchHTML: Content-Encoding: '%s'", contentEncoding)
			
			var reader io.Reader = resp.Body
			
			// Descomprimir baseado no Content-Encoding
			// O Go HTTP client normalmente descomprime automaticamente, mas vamos garantir
			contentEncodingLower := strings.ToLower(strings.TrimSpace(contentEncoding))
			switch contentEncodingLower {
			case "gzip":
				log.Printf("[Scraper] FetchHTML: Descomprimindo com gzip...")
				gzReader, err := gzip.NewReader(resp.Body)
				if err != nil {
					log.Printf("[Scraper] FetchHTML: ERRO ao criar gzip reader: %v", err)
					resp.Body.Close()
					transport.CloseIdleConnections()
					lastErr = err
					if proxyManager.HasProxies() && retry < maxRetries-1 {
						proxyURL = proxyManager.GetNextProxy()
						continue
					} else {
						proxyURL = ""
						continue
					}
				}
				reader = gzReader
			case "br", "brotli":
				log.Printf("[Scraper] FetchHTML: Descomprimindo com brotli...")
				brReader := brotli.NewReader(resp.Body)
				reader = brReader
			case "deflate":
				log.Printf("[Scraper] FetchHTML: Content-Encoding deflate detectado (não suportado, tentando sem descompressão)")
			case "":
				// Sem Content-Encoding header, mas pode estar comprimido mesmo assim
				// Vamos tentar detectar se está comprimido pelos primeiros bytes
				log.Printf("[Scraper] FetchHTML: Sem Content-Encoding header, verificando magic numbers...")
				// Ler primeiros bytes para verificar
				peekBytes := make([]byte, 2)
				peekReader := io.LimitReader(resp.Body, 2)
				n, _ := peekReader.Read(peekBytes)
				if n >= 2 {
					// Verificar magic numbers: gzip (1f 8b) ou brotli
					if peekBytes[0] == 0x1f && peekBytes[1] == 0x8b {
						log.Printf("[Scraper] FetchHTML: Detectado gzip pelos magic numbers (0x%02x 0x%02x), descomprimindo...", peekBytes[0], peekBytes[1])
						// Recriar reader com os bytes já lidos + resto
						bodyReader := io.MultiReader(strings.NewReader(string(peekBytes[:n])), resp.Body)
						gzReader, err := gzip.NewReader(bodyReader)
						if err == nil {
							reader = gzReader
						} else {
							log.Printf("[Scraper] FetchHTML: ERRO ao criar gzip reader: %v, usando sem descompressão", err)
							reader = io.MultiReader(strings.NewReader(string(peekBytes[:n])), resp.Body)
						}
					} else {
						log.Printf("[Scraper] FetchHTML: Magic numbers (0x%02x 0x%02x) não indicam compressão conhecida", peekBytes[0], peekBytes[1])
						// Não é gzip, recriar reader
						reader = io.MultiReader(strings.NewReader(string(peekBytes[:n])), resp.Body)
					}
				} else {
					log.Printf("[Scraper] FetchHTML: Poucos bytes para detectar compressão (%d bytes)", n)
					// Poucos bytes, recriar reader
					reader = io.MultiReader(strings.NewReader(string(peekBytes[:n])), resp.Body)
				}
			default:
				log.Printf("[Scraper] FetchHTML: Compressão desconhecida: '%s'", contentEncoding)
			}
			
			// Ler todo o HTML em memória para debug e depois retornar
			// (páginas HTML geralmente são pequenas, então é seguro)
			allBodyBytes, err := io.ReadAll(reader)
			if err != nil {
				log.Printf("[Scraper] FetchHTML: ERRO ao ler body completo: %v", err)
				resp.Body.Close()
				transport.CloseIdleConnections()
				lastErr = err
				if proxyManager.HasProxies() && retry < maxRetries-1 {
					proxyURL = proxyManager.GetNextProxy()
					continue
				} else {
					proxyURL = ""
					continue
				}
			}
			resp.Body.Close()
			
			htmlContent := string(allBodyBytes)
			log.Printf("[Scraper] FetchHTML: HTML recebido completo (%d bytes)", len(allBodyBytes))
			
			// Verificar se contém a classe que estamos procurando
			if strings.Contains(htmlContent, "result__photoLink") {
				log.Printf("[Scraper] FetchHTML: ✓ Classe 'result__photoLink' encontrada no HTML")
				// Encontrar todas as ocorrências
				count := strings.Count(htmlContent, "result__photoLink")
				log.Printf("[Scraper] FetchHTML: Classe 'result__photoLink' aparece %d vezes no HTML", count)
				// Mostrar primeira ocorrência com contexto
				idx := strings.Index(htmlContent, "result__photoLink")
				if idx >= 0 {
					start := idx - 150
					if start < 0 {
						start = 0
					}
					end := idx + 300
					if end > len(htmlContent) {
						end = len(htmlContent)
					}
					log.Printf("[Scraper] FetchHTML: Primeira ocorrência (contexto): ...%s...", htmlContent[start:end])
				}
			} else {
				log.Printf("[Scraper] FetchHTML: ✗ Classe 'result__photoLink' NÃO encontrada no HTML")
				// Procurar por variações
				if strings.Contains(htmlContent, "result") {
					log.Printf("[Scraper] FetchHTML: Mas contém 'result', procurando variações...")
					// Contar ocorrências de "result"
					resultCount := strings.Count(htmlContent, "result")
					log.Printf("[Scraper] FetchHTML: 'result' aparece %d vezes no HTML", resultCount)
					// Procurar por "result" e mostrar contexto
					idx := strings.Index(htmlContent, "result")
					if idx >= 0 {
						start := idx - 100
						if start < 0 {
							start = 0
						}
						end := idx + 200
						if end > len(htmlContent) {
							end = len(htmlContent)
						}
						log.Printf("[Scraper] FetchHTML: Primeira ocorrência de 'result' (contexto): ...%s...", htmlContent[start:end])
					}
					// Procurar por "photoLink" ou variações
					if strings.Contains(htmlContent, "photoLink") {
						log.Printf("[Scraper] FetchHTML: Encontrado 'photoLink' (sem 'result__')")
					}
					if strings.Contains(htmlContent, "photo-link") {
						log.Printf("[Scraper] FetchHTML: Encontrado 'photo-link' (com hífen)")
					}
				}
				// Mostrar primeiros 1000 caracteres para debug
				previewLen := 1000
				if len(htmlContent) < previewLen {
					previewLen = len(htmlContent)
				}
				log.Printf("[Scraper] FetchHTML: Primeiros %d caracteres do HTML: %s", previewLen, htmlContent[:previewLen])
			}
			
			// Fechar conexões idle antes de retornar com sucesso
			transport.CloseIdleConnections()
			log.Printf("[Scraper] FetchHTML: retornando body com sucesso")
			return io.NopCloser(strings.NewReader(htmlContent)), nil
		}

		// Se chegou aqui, este proxy falhou, tentar próximo
		log.Printf("[Scraper] FetchHTML: Proxy falhou, tentando próximo...")
		// Fechar conexões idle antes de tentar próximo proxy
		transport.CloseIdleConnections()
		
		if proxyManager.HasProxies() && retry < maxRetries-1 {
			proxyURL = proxyManager.GetNextProxy() // Rotaciona para próxima tentativa
		} else {
			proxyURL = "" // Tentar sem proxy no final
		}
	}

	// Se chegou aqui, todos os proxies falharam
	// Fechar qualquer conexão restante
	if lastErr != nil {
		return nil, fmt.Errorf("all proxies failed, last error: %v", lastErr)
	}
	return nil, fmt.Errorf("all proxies failed")
}
