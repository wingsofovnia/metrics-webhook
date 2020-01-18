package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
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
	log.Printf("Listening and serving on %s", l.server.Addr)
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
	server.ListenAndServe()
}
