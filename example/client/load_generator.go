package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	genClients := flag.Int("increment", 100, "Amount of concurrent clients to add up with every generation (default: 100)")
	genDuration := flag.Duration("duration", 2*time.Minute, "The duration of each generation. Zero means keep the load constant with $increment clients. (default: 2m)")
	clientSleep := flag.Duration("sleep", 1*time.Second, "Delay between a call and re-enqueue. (default: 1s)")
	server := flag.String("server", "http://localhost:8080", "Url to send requests to (default: http://localhost:8080)")
	method := flag.String("method", "GET", "HTTP request method (default: GET)")
	nobody := flag.Bool("nobody", false, "Ignores response body (default: false)")
	maxbody := flag.Int64("maxbody", -1, "Amount of bytes to read from response if nobody = false, -1 = full (default: -1)")
	flag.Parse()

	// Display config
	fmt.Println("Load Generator Configuration:")
	fmt.Printf(" - Request = %s %s\n", *method, *server)
	if *genDuration > 0 {
		fmt.Printf(" - Concurrent requests: %d + %d every generation\n", *genClients, *genClients)
		fmt.Printf(" - New generation spawn: every %s\n", genDuration.String())
	} else {
		fmt.Printf(" - Concurrent requests: %d\n", *genClients)
	}
	fmt.Printf(" - Re-enqueue delay: %s\n", genDuration.String())
	fmt.Printf(" - Fetch body: %t\n", !*nobody)
	if *maxbody > 0 {
		fmt.Printf(" - Body limitation = %d bytes\n", *maxbody)
	}
	fmt.Println()

	// Channel used for stopping clients and generations
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	signal.Notify(interrupt, os.Kill)

	// Signals creation of a new generation
	genPromoter := make(chan time.Time)

	// Setup generation Ticker if enabled
	if *genDuration > 0 {
		genTicker := time.NewTicker(*genDuration)
		go func() {
			for {
				select {
				case t := <-genTicker.C:
					genPromoter <- t
				case <-interrupt:
					genTicker.Stop()
					return
				}
			}
		}()
	}

	// Trigger the first generation (an possible the only, if Ticker not set up)
	go func() {
		genPromoter <- time.Now()
	}()

	// A queue of client requests. 1 signal = 1 call
	clients := make(chan bool)
	go func() {
		for {
			select {
			case genTime := <-genPromoter:
				fmt.Printf("Generation %s: +%d clients arrived\n", genTime.Format("2 Jan 2006 15:04:05"), *genClients)
				for c := 0; c < *genClients; c++ {
					go func() {
						clients <- true
					}()
				}
			case <-interrupt:
				return
			}
		}
	}()

	for {
		select {
		case <-clients:
			go func() {
				// Apply pressure on the server
				pressure(*server, *method, *nobody, *maxbody)

				// Cooling down
				time.Sleep(*clientSleep)

				// Re-enqueue client again
				go func() {
					clients <- true
				}()
			}()
		case <-interrupt:
			return
		}
	}
}

func pressure(server, method string, nobody bool, maxbody int64) {
	req, err := http.NewRequest(method, server, nil)
	if err != nil {
		fmt.Printf("failed to create a request: %+v\n", err)
		return
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to send a request: %+v\n", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 || res.StatusCode < 200 {
		fmt.Printf("server responded non 2xx: HTTP %d\n", res.StatusCode)
		return
	}

	if !nobody {
		bodyReader := io.Reader(res.Body)
		if maxbody > 0 {
			bodyReader = io.LimitReader(res.Body, maxbody)
		}
		_, err := io.Copy(ioutil.Discard, bodyReader)
		if err != nil {
			fmt.Printf("failed to read body: %+v\n", err)
			return
		}
	}
}
