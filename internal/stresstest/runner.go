package stresstest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

// Config define os parâmetros de uma execução do teste de carga.
type Config struct {
	URL         string
	Requests    int
	Concurrency int
	Timeout     time.Duration
}

// Validate verifica se a configuração é utilizável.
func (c Config) Validate() error {
	if c.URL == "" {
		return errors.New("--url é obrigatório")
	}
	if c.Requests <= 0 {
		return errors.New("--requests deve ser maior que zero")
	}
	if c.Concurrency <= 0 {
		return errors.New("--concurrency deve ser maior que zero")
	}
	return nil
}

// Report consolida o resultado da execução.
type Report struct {
	TotalDuration time.Duration
	TotalRequests int
	StatusCodes   map[int]int    // status HTTP -> quantidade
	Errors        map[string]int // erro de transporte (timeout, conexão recusada...) -> quantidade
}

// OK retorna a quantidade de respostas com status 200.
func (r Report) OK() int { return r.StatusCodes[http.StatusOK] }

// ErrorCount retorna a quantidade de requisições que falharam sem resposta HTTP.
func (r Report) ErrorCount() int {
	total := 0
	for _, n := range r.Errors {
		total += n
	}
	return total
}

const maxRedirects = 10

type result struct {
	status int
	err    error
}

// Run executa exatamente cfg.Requests requisições GET, com no máximo
// cfg.Concurrency requisições simultâneas.
func Run(ctx context.Context, cfg Config) (Report, error) {
	if err := cfg.Validate(); err != nil {
		return Report{}, err
	}
	if _, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil); err != nil {
		return Report{}, err
	}

	workers := min(cfg.Concurrency, cfg.Requests)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = workers
	transport.MaxIdleConnsPerHost = workers
	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
		// Em loops de redirecionamento, contabiliza o status da última resposta
		// em vez de tratar a requisição como falha sem status HTTP.
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	defer transport.CloseIdleConnections()

	jobs := make(chan struct{}, cfg.Requests)
	for range cfg.Requests {
		jobs <- struct{}{}
	}
	close(jobs)

	results := make(chan result, workers)
	var wg sync.WaitGroup

	start := time.Now()
	for range workers {
		wg.Go(func() {
			for range jobs {
				results <- doRequest(ctx, client, cfg.URL)
			}
		})
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	report := Report{StatusCodes: map[int]int{}, Errors: map[string]int{}}
	for res := range results {
		report.TotalRequests++
		if res.err != nil {
			report.Errors[classifyError(res.err)]++
			continue
		}
		report.StatusCodes[res.status]++
	}
	report.TotalDuration = time.Since(start)
	return report, nil
}

func doRequest(ctx context.Context, client *http.Client, url string) result {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return result{err: err}
	}
	resp, err := client.Do(req)
	if err != nil {
		return result{err: err}
	}
	defer resp.Body.Close()
	// Consome o corpo para permitir a reutilização da conexão (keep-alive).
	_, _ = io.Copy(io.Discard, resp.Body)
	return result{status: resp.StatusCode}
}

func classifyError(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelado"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	}
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	return "erro de conexão"
}
