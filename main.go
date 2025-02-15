package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
)

type errorReturn struct {
	ErrorMessage string `json:"error"`
}

type chirpValidReturn struct {
	ValidBool bool `json:"valid"`
}

type chirpStringReturn struct {
	CleanedBody string `json:"cleaned_body"`
}

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

func returnPlainTextContent(rw http.ResponseWriter) {
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
}

func returnHtmlContent(rw http.ResponseWriter) {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
}

func returnJsonContent(rw http.ResponseWriter, statusCode int, er *errorReturn, ok *chirpStringReturn) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(statusCode)

	if er != nil {
		data, err := json.Marshal(er)
		if err != nil {
			rw.Write([]byte("Error kod json.Marshala za error message koji bi trebalo vratiti. " + err.Error()))

		} else {
			rw.Write(data)
		}
	}

	if ok != nil {
		data, err := json.Marshal(ok)
		if err != nil {
			rw.Write([]byte("Error kod json.Marshala za chirpValidReturn. " + err.Error()))
		} else {
			rw.Write(data)
		}
	}
}

func (cfg *apiConfig) handleMetrics_TextFormat(rw http.ResponseWriter, req *http.Request) {
	returnPlainTextContent(rw)
	var getInt int32 = cfg.fileServerHits.Load()
	var output string = "Hits: " + strconv.Itoa(int(getInt))
	rw.Write([]byte(output))
}

func (cfg *apiConfig) handleMetrics(rw http.ResponseWriter, req *http.Request) {
	var rawTemplate string = `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`

	returnHtmlContent(rw)
	var getInt int32 = cfg.fileServerHits.Load()
	var output string = fmt.Sprintf(rawTemplate, int(getInt))
	rw.Write([]byte(output))
}

func (cfg *apiConfig) handleReset(rw http.ResponseWriter, req *http.Request) {
	returnPlainTextContent(rw)
	cfg.fileServerHits.Store(0)
}

// type healthHandler struct{}

func /* (handler healthHandler) */ handleHealth(rw http.ResponseWriter, req *http.Request) {
	returnPlainTextContent(rw)
	rw.Write([]byte("OK"))
}

func cleanString(inputstr string) string {
	var nedopusteni []string = []string{
		strings.ToUpper("kerfuffle"),
		strings.ToUpper("sharbert"),
		strings.ToUpper("fornax"),
	}

	var tokeni []string = strings.Split(inputstr, " ")
	for i := 0; i < len(tokeni); i++ {
		var token = strings.ToUpper(tokeni[i])
		for j := 0; j < len(nedopusteni); j++ {
			var invalid_token = nedopusteni[j]
			if invalid_token == token {
				tokeni[i] = "****"
				break
			}
		}
	}

	return strings.Join(tokeni, " ")
}

func validateChirp(rw http.ResponseWriter, req *http.Request) {
	type parameters struct {
		BodyText string `json:"body"`
	}
	er_return := errorReturn{}

	if req.ContentLength == 0 {
		er_return.ErrorMessage = "Something went wrong"
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	// params ima .body iz requesta
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	// vratimo json error ako je failan parse
	if err != nil {
		er_return.ErrorMessage = "Something went wrong"
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	if len(params.BodyText) == 0 {
		er_return.ErrorMessage = "Something went wrong"
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	if len(params.BodyText) > 140 {
		er_return.ErrorMessage = "Chirp is too long"
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	// here
	var cleanBody string = cleanString(params.BodyText)
	var string_return chirpStringReturn
	string_return.CleanedBody = cleanBody
	returnJsonContent(rw, http.StatusOK, nil, &string_return)

	/*
		var valid_return chirpValidReturn
		valid_return.ValidBool = true
		returnJsonContent(rw, http.StatusOK, nil, &valid_return)
	*/
}

func main() {
	// Setup dio:
	var port string = "8080"
	mux := http.NewServeMux() // *ServeMux

	var srv *http.Server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// API specification dio:
	var apiCfg apiConfig
	mux.HandleFunc("/api/healthz", handleHealth)
	mux.HandleFunc("/api/validate_chirp", validateChirp)

	// "Admin" dio:
	mux.HandleFunc("/admin/reset", apiCfg.handleReset)
	mux.HandleFunc("/admin/metrics", apiCfg.handleMetrics)

	// Web specification dio:
	var file_server http.Handler = http.FileServer(http.Dir("."))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", file_server)))

	// Startup dio:
	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}
