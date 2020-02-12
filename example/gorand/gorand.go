package main

import (
	"fmt"
	"math"
	"net/http"

	"github.com/wingsofovnia/metrics-webhook/lib"
	"github.com/wingsofovnia/metrics-webhook/pkg/apis/metrics/v1alpha1"

	log "github.com/sirupsen/logrus"
)

type GorandServer struct {
	server    *http.Server
	randChars int32
}

func NewGorandServer(addr string, randChars int32) *GorandServer {
	router := http.NewServeMux()
	server := &GorandServer{
		server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
		randChars: randChars,
	}

	router.HandleFunc("/", server.writeRand)
	return server
}

func NewDefaultLoremServer() *GorandServer {
	return NewGorandServer(":8080", math.MaxInt16)
}

func (l *GorandServer) ListenAndServe() error {
	log.Infof("Listening and serving on %s", l.server.Addr)
	return l.server.ListenAndServe()
}

func (l *GorandServer) writeRand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, RandString(l.randChars))
}

const randCharsConfig = "RAND_CHARS"
const randCharsInit = math.MaxInt16
const randCharsMin = 5000
const randCharsFallbackAdj = -2457

const correlatorBufferFlushCap = 3
const correlatorOvershoot = 0.1

func main() {
	log.Infof("Gorand config: {init = %d, min = %d, fallback = %d, correlator.buffercap = %d, correlator.overshoot = %.2f}",
		randCharsInit, randCharsMin, randCharsFallbackAdj, correlatorBufferFlushCap, correlatorOvershoot)

	// Random string generator server
	server := NewGorandServer(":8080", randCharsInit)

	// Adjustment SDK
	correlator, _ := lib.NewAdjustmentCorrelator(3, 0.10)
	var alertCallback = func(report v1alpha1.MetricReport) {
		adjustments := make(lib.Adjustments)

		if server.randChars <= randCharsMin {
			log.Warnf("Gorand reached the lower possible randChar config, no adjustments applied")
			return
		}

		log.Infof("Incoming notifications: %s", report.String())

		if report.HasAlerts() {
			suggestions := correlator.SuggestAdjustments(report)

			if loremSuggestion, set := suggestions[randCharsConfig]; set {
				log.Infof("Correlator suggests %s = %f", randCharsConfig, loremSuggestion)
				adjustments[randCharsConfig] = loremSuggestion
			} else {
				log.Infof("Correlator gave no suggestions, default adjustment %s = %d", randCharsConfig, randCharsFallbackAdj)
				adjustments[randCharsConfig] = float64(randCharsFallbackAdj)
			}

			prevLoremChars := server.randChars
			server.randChars = max(randCharsMin, server.randChars + int32(adjustments[randCharsConfig]))

			log.Infof("%s has been adjusted (was = %d, adjustment = %f, now = %d)",
				randCharsConfig, prevLoremChars, adjustments[randCharsConfig], server.randChars)
		} else {
			log.Infoln("No alerts present, no adjustments has been made.")
		}

		correlator.RegisterAdjustments(report, adjustments)
	}

	// Metrics Webhook server
	webhookServer := lib.NewWebhookServer(alertCallback)
	go func() {
		webhookServer.ListenAndServe()
	}()

	server.ListenAndServe()
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
