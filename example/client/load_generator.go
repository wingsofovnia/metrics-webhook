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
	clients := flag.Int("clients", 100, "Amount of concurrent clients (default: 10)")
	server := flag.String("server", "https://google.com", "Url to send requests to (default: https://google.com)")
	method := flag.String("method", "GET", "HTTP request method (default: GET)")
	nobody := flag.Bool("nobody", false, "Ignores response body (default: false)")
	maxbody := flag.Int64("maxbody", -1, "Amount of bytes to read from response if nobody = false, -1 = full (default: -1)")
	clientSync := flag.Bool("sync", true, "Enables client request synchronization for a more smooth load (default: true)")
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	})

	log.Infoln("Load Generator Configuration:")
	log.Infof(" - clients = %d", *clients)
	log.Infof(" - server = %s", *server)
	log.Infof(" - method = %s", *method)
	log.Infof(" - nobody = %t", *nobody)
	log.Infof(" - maxbody = %d", *maxbody)
	log.Infof(" - client synchronization = %t", *clientSync)

	var wg sync.WaitGroup
	for g := 1; ; g++ {
		log.Infof("Generation #%d\n", g)
		bar := pb.StartNew(*clients)
		for c := 0; c < *clients; c++ {
			if *clientSync {
				wg.Add(1)
			}
			go func(clientId int) {
				defer bar.Increment()
				if *clientSync {
					defer func() {
						wg.Done()
					}()
				}

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
