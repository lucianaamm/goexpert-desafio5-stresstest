package stresstest

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

// Print escreve o relatório formatado em w.
func (r Report) Print(w io.Writer, url string) {
	line := strings.Repeat("=", 50)
	fmt.Fprintln(w, line)
	fmt.Fprintln(w, "              RELATÓRIO DO TESTE DE CARGA")
	fmt.Fprintln(w, line)
	fmt.Fprintf(w, "URL:                         %s\n", url)
	fmt.Fprintf(w, "Tempo total:                 %s\n", r.TotalDuration.Round(1e6))
	fmt.Fprintf(w, "Total de requests:           %d\n", r.TotalRequests)
	fmt.Fprintf(w, "Requests com status 200:     %d\n", r.OK())
	if secs := r.TotalDuration.Seconds(); secs > 0 {
		fmt.Fprintf(w, "Requests por segundo:        %.2f\n", float64(r.TotalRequests)/secs)
	}

	codes := make([]int, 0, len(r.StatusCodes))
	for code := range r.StatusCodes {
		if code != http.StatusOK {
			codes = append(codes, code)
		}
	}
	slices.Sort(codes)

	fmt.Fprintln(w, line)
	fmt.Fprintln(w, "Distribuição de outros status HTTP:")
	if len(codes) == 0 {
		fmt.Fprintln(w, "  (nenhum)")
	}
	for _, code := range codes {
		fmt.Fprintf(w, "  %d %-24s %d\n", code, http.StatusText(code), r.StatusCodes[code])
	}

	if len(r.Errors) > 0 {
		kinds := make([]string, 0, len(r.Errors))
		for k := range r.Errors {
			kinds = append(kinds, k)
		}
		slices.Sort(kinds)
		fmt.Fprintln(w, line)
		fmt.Fprintf(w, "Falhas sem resposta HTTP:    %d\n", r.ErrorCount())
		for _, k := range kinds {
			fmt.Fprintf(w, "  %-28s %d\n", k, r.Errors[k])
		}
	}
	fmt.Fprintln(w, line)
}
