package main

import (
	"log"
	"net/http"
)

// build cmd: go build -o out && ./out
// curl -o chirpy-logo.png https://storage.googleapis.com/qvault-webapp-dynamic-assets/course_assets/2CofkLc.png

type healthHandler struct{}

func (handler healthHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("OK"))
}

func main() {
	var port string = "8080"
	mux := http.NewServeMux() // *ServeMux

	var srv *http.Server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	var file_server http.Handler = http.FileServer(http.Dir("."))

	mux.Handle("/healthz", healthHandler{})
	// mux.Handle("/", file_server)
	mux.Handle("/app/", http.StripPrefix("/app", file_server))

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}
