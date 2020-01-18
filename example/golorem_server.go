package main

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/wingsofovnia/metrics-webhook/lib"
	"github.com/wingsofovnia/metrics-webhook/pkg/apis/metrics/v1alpha1"
)

type LoremServer struct {
	server     *http.Server
	loremChars int
}

func NewLoremServer(addr string, loremChars int) *LoremServer {
	router := http.NewServeMux()
	server := &LoremServer{
		server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
		loremChars: loremChars,
	}

	router.HandleFunc("/", server.writeLorem)
	return server
}

func NewDefaultLoremServer() *LoremServer {
	return NewLoremServer(":8080", len(Lipsum)*3)
}

func (l *LoremServer) ListenAndServe() error {
	log.Infof("[GoLorem] Listening and serving on %s", l.server.Addr)
	return l.server.ListenAndServe()
}

func (l *LoremServer) writeLorem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, l.lorem())
}

func (l *LoremServer) lorem() string {
	return strings.Join(Lorem(l.loremChars), "\n")
}

func main() {
	server := NewDefaultLoremServer()
	go func() {
		const loremConfig = "lorem"
		const loremConfigDefault = -len(Lipsum)

		correlator := lib.NewDefaultAdjustmentCorrelator()

		webhookServer := lib.NewWebhookServer(lib.DefaultWebhookServerConfig(), func(report v1alpha1.MetricReport) {
			adjustments := make(lib.Adjustments)

			if report.HasAlerts() {
				suggestions := correlator.SuggestAdjustments(report)
				if loremSuggestion, set := suggestions[loremConfig]; set {
					adjustments[loremConfig] = loremSuggestion
				} else {
					adjustments[loremConfig] = float64(loremConfigDefault)
				}

				server.loremChars = server.loremChars + int(adjustments[loremConfig])
			}

			correlator.RegisterAdjustments(report, adjustments)
		})

		webhookServer.ListenAndServe()
	}()

	server.ListenAndServe()
}
