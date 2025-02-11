package main

import (
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
)

// build cmd: go build -o out && ./out
// curl -o chirpy-logo.png https://storage.googleapis.com/qvault-webapp-dynamic-assets/course_assets/2CofkLc.png

type apiConfig struct {
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func returnPlainText(rw http.ResponseWriter) {
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
}

func (cfg *apiConfig) handleMetrics(rw http.ResponseWriter, req *http.Request) {
	returnPlainText(rw)
	var getInt int32 = cfg.fileServerHits.Load()
	var output string = "Hits: " + strconv.Itoa(int(getInt))
	rw.Write([]byte(output))
}

func (cfg *apiConfig) handleReset(rw http.ResponseWriter, req *http.Request) {
	returnPlainText(rw)
	cfg.fileServerHits.Store(0)
}

// type healthHandler struct{}

func /* (handler healthHandler) */ handleHealth(rw http.ResponseWriter, req *http.Request) {
	returnPlainText(rw)
	rw.Write([]byte("OK"))
}

func main() {
	var apiCfg apiConfig
	// var health_instance healthHandler

	var port string = "8080"
	mux := http.NewServeMux() // *ServeMux

	var srv *http.Server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	var file_server http.Handler = http.FileServer(http.Dir("."))

	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/metrics", apiCfg.handleMetrics)
	mux.HandleFunc("/reset", apiCfg.handleReset)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", file_server)))

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}
