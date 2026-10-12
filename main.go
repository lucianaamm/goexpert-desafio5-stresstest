package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucianaamm/goexpert-desafio5-stresstest/internal/stresstest"
)

func main() {
	var cfg stresstest.Config
	flag.StringVar(&cfg.URL, "url", "", "URL do serviço a ser testado")
	flag.IntVar(&cfg.Requests, "requests", 0, "número total de requisições")
	flag.IntVar(&cfg.Concurrency, "concurrency", 1, "número de chamadas simultâneas")
	flag.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "timeout de cada requisição")
	flag.Parse()

	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Iniciando teste: %d requests, concorrência %d -> %s\n\n", cfg.Requests, cfg.Concurrency, cfg.URL)
	report, err := stresstest.Run(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	report.Print(os.Stdout, cfg.URL)
}
