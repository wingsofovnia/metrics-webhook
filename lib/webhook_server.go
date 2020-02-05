package lib

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/wingsofovnia/metrics-webhook/pkg/apis/metrics/v1alpha1"
)

type Webhook func(report v1alpha1.MetricReport)

var WebhookHandler = func(callback Webhook) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		decoder := json.NewDecoder(r.Body)
		var report v1alpha1.MetricReport
		err := decoder.Decode(&report)
		if err != nil {
			log.Printf("[Webhook] ERR: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		callback(report)
	}
}

type WebhookServer struct {
	httpServer *http.Server
}

type WebhookServerConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	WebhookPath  string
}

func DefaultWebhookServerConfig() *WebhookServerConfig {
	return &WebhookServerConfig{
		Addr:         ":4030",
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
		IdleTimeout:  time.Second * 60,
		WebhookPath:  "/metrics-webhook",
	}
}

func NewWebhookServer(cfg *WebhookServerConfig, callback Webhook) *WebhookServer {
	if cfg == nil {
		cfg = DefaultWebhookServerConfig()
	}

	router := http.NewServeMux()
	router.HandleFunc(cfg.WebhookPath, WebhookHandler(callback))

	return &WebhookServer{
		httpServer: &http.Server{
			Addr:         cfg.Addr,
			WriteTimeout: cfg.WriteTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			IdleTimeout:  cfg.IdleTimeout,
			Handler:      router,
		},
	}
}

func (srv *WebhookServer) ListenAndServe() {
	log.Printf("[Webhook] Listening and serving on %s", srv.httpServer.Addr)
	go func() {
		if err := srv.httpServer.ListenAndServe(); err != nil {
			log.Printf("[Webhook] ERR: %v", err)
		}
	}()
}

func (srv *WebhookServer) Shutdown(ctx context.Context) error {
	return srv.httpServer.Shutdown(ctx)
}
