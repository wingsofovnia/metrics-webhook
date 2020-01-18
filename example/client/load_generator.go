package main

import (
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/cheggaaa/pb/v3"
	log "github.com/sirupsen/logrus"
)

func main() {
	clientsInit := flag.Int("clients", 10, "Amount of concurrent clients in the first generation (default: 100)")
	clientsIncr := flag.Int("increment", 5, "Amount of clients to add up with every generation (default: 0)")
	server := flag.String("server", "https://www.google.com/maps", "Url to send requests to (default: https://google.com)")
	method := flag.String("method", "GET", "HTTP request method (default: GET)")
	nobody := flag.Bool("nobody", false, "Ignores response body (default: false)")
	maxbody := flag.Int64("maxbody", -1, "Amount of bytes to read from response if nobody = false, -1 = full (default: -1)")
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	})

	log.Infoln("Load Generator Configuration:")
	log.Infof(" - Request = %s %s", *method, *server)
	if *clientsIncr <= 0 {
		log.Infof(" - Concurrent requests: %d", *clientsInit)
	} else {
		log.Infof(" - Concurrent requests: %d with +%d increment", *clientsInit, *clientsIncr)
	}
	log.Infof(" - Fetch body: %t", *nobody)
	if *maxbody > 0 {
		log.Infof(" - Body limitation = %d bytes", *maxbody)
	}

	var wg sync.WaitGroup
	for g := 1; ; g++ {
		clients := *clientsInit + *clientsIncr*(g-1)
		bar := pb.StartNew(clients)
		for c := 1; c <= clients; c++ {
			wg.Add(1)
			go func(clientId int) {
				defer func() {
					bar.Increment()
					wg.Done()
				}()

				req, err := http.NewRequest(*method, *server, nil)
				if err != nil {
					log.Errorf("Client #%d: failed to create a request: %+v", clientId, err)
					return
				}
				res, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Errorf("Client #%d: failed to send a request: %+v", clientId, err)
					return
				}
				defer res.Body.Close()
				if res.StatusCode >= 300 || res.StatusCode < 200 {
					log.Errorf("Client #%d: server responded non 2xx: HTTP %d", clientId, res.StatusCode)
					return
				}

				if !*nobody {
					bodyReader := io.Reader(res.Body)
					if *maxbody > 0 {
						bodyReader = io.LimitReader(res.Body, *maxbody)
					}
					_, err := io.Copy(ioutil.Discard, bodyReader)
					if err != nil {
						log.Errorf("Client #%d: failed to read body: %+v", clientId, err)
						return
					}
				}
			}(c)
		}
		wg.Wait()
		bar.Finish()
	}
}
