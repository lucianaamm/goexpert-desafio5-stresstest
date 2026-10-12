package stresstest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunExactRequestCountAndStatusDistribution(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		switch {
		case n%10 == 0:
			w.WriteHeader(http.StatusInternalServerError)
		case n%5 == 0:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	report, err := Run(context.Background(), Config{URL: srv.URL, Requests: 100, Concurrency: 7, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if got := hits.Load(); got != 100 {
		t.Fatalf("servidor recebeu %d requests, esperado 100", got)
	}
	if report.TotalRequests != 100 {
		t.Fatalf("TotalRequests = %d, esperado 100", report.TotalRequests)
	}
	if report.OK() != 80 || report.StatusCodes[404] != 10 || report.StatusCodes[500] != 10 {
		t.Fatalf("distribuição inesperada: %v", report.StatusCodes)
	}
}

func TestRunRespectsConcurrency(t *testing.T) {
	var inFlight, peak atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			p := peak.Load()
			if cur <= p || peak.CompareAndSwap(p, cur) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}))
	defer srv.Close()

	if _, err := Run(context.Background(), Config{URL: srv.URL, Requests: 40, Concurrency: 4, Timeout: 5 * time.Second}); err != nil {
		t.Fatal(err)
	}
	if p := peak.Load(); p > 4 {
		t.Fatalf("pico de concorrência %d excede 4", p)
	}
}

func TestRunCountsTransportErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // ninguém escutando: todas as requisições falham

	report, err := Run(context.Background(), Config{URL: url, Requests: 5, Concurrency: 2, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalRequests != 5 || report.ErrorCount() != 5 {
		t.Fatalf("esperado 5 falhas de 5 requests, obtido %+v", report)
	}
}

func TestRunRedirectLoopReportsLastStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	}))
	defer srv.Close()

	report, err := Run(context.Background(), Config{URL: srv.URL, Requests: 3, Concurrency: 1, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if report.StatusCodes[http.StatusFound] != 3 || report.ErrorCount() != 0 {
		t.Fatalf("esperado 3x 302 sem falhas, obtido %+v", report)
	}
}

func TestConfigValidate(t *testing.T) {
	cases := []Config{
		{Requests: 1, Concurrency: 1},
		{URL: "http://x", Requests: 0, Concurrency: 1},
		{URL: "http://x", Requests: 1, Concurrency: 0},
	}
	for _, c := range cases {
		if c.Validate() == nil {
			t.Errorf("esperado erro para %+v", c)
		}
	}
}
