package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/ihladni/chirpy/internal/auth"
	"github.com/ihladni/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

/*
The underscore import (_ "package/name") is used when you need to import a package for its side effects only,
but don't directly use any of its exported names.

Common examples include:
    Database drivers (like _ "github.com/lib/pq" for PostgreSQL)
    Packages that register themselves on init()
    Packages that need to run their init() functions
*/

type MappedChirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    string    `json:"user_id"`
}

func DBChirpToMappedChirp(dbChirp database.Chirp) MappedChirp {
	var returnMappedChirp MappedChirp

	returnMappedChirp.ID = dbChirp.ID
	returnMappedChirp.CreatedAt = dbChirp.CreatedAt
	returnMappedChirp.UpdatedAt = dbChirp.UpdatedAt
	returnMappedChirp.Body = dbChirp.Body

	if dbChirp.UserID.Valid {
		returnMappedChirp.UserID = dbChirp.UserID.UUID.String()
	} else {
		returnMappedChirp.UserID = ""
	}
	return returnMappedChirp
}

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
	db             *database.Queries
	platform       string
	fileServerHits atomic.Int32
}

type parameters struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type MappedUser struct {
	ID             uuid.UUID `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Email          string    `json:"email"`
	HashedPassword string    `json:"-"` // The "-" means "don't include this in JSON"
}

// ---------------------------------------------------------------------------------------

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
	if cfg.platform == "dev" {
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	// returnPlainTextContent(rw)

	cfg.db.DeleteUsers(req.Context())

	cfg.fileServerHits.Store(0)

	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
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

// POST /api/login
func (cfg *apiConfig) loginHandler(rw http.ResponseWriter, req *http.Request) {
	er_return := errorReturn{}
	params := parameters{}

	if req.ContentLength == 0 {
		er_return.ErrorMessage = "Something went wrong. No content received."
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	decoder := json.NewDecoder(req.Body)

	err := decoder.Decode(&params)
	if err != nil {
		er_return.ErrorMessage = "Something went wrong. Error on Decode."
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	dbUser, err := cfg.db.GetUserByEmail(req.Context(), params.Email)
	if err != nil {
		er_return.ErrorMessage = "No user."
		returnJsonContent(rw, http.StatusUnauthorized, &er_return, nil)
		return
	}

	err = auth.CheckPasswordHash(params.Password, dbUser.HashedPassword)
	if err != nil {
		er_return.ErrorMessage = "Incorrect email or password"
		returnJsonContent(rw, http.StatusUnauthorized, &er_return, nil)
		return
	}

	var returnUser MappedUser
	returnUser.ID = dbUser.ID
	returnUser.CreatedAt = dbUser.CreatedAt
	returnUser.UpdatedAt = dbUser.UpdatedAt
	returnUser.Email = dbUser.Email

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	data, err := json.Marshal(returnUser)
	if err != nil {
		rw.Write([]byte("Error kod json.Marshala za jsonMarshal za returnUser objekt. " + err.Error()))
	} else {
		rw.Write(data)
	}

}

// POST /api/users
func (cfg *apiConfig) createUser(rw http.ResponseWriter, req *http.Request) {

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
		er_return.ErrorMessage = "Something went wrong. Error on Decode."
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	if len(params.Email) == 0 {
		er_return.ErrorMessage = "Something went wrong. No email."
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	var create_args database.CreateUserParams
	create_args.Email = params.Email

	hpwd, err := auth.HashPassword(params.Password)
	if err != nil {
		er_return.ErrorMessage = "Something went wrong. HashPassword error."
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}
	create_args.HashedPassword = hpwd

	dbUser, err := cfg.db.CreateUser(req.Context(), create_args)
	if err != nil {
		er_return.ErrorMessage = "Something went wrong. CreateUser error."
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	var returnUser MappedUser
	returnUser.ID = dbUser.ID
	returnUser.CreatedAt = dbUser.CreatedAt
	returnUser.UpdatedAt = dbUser.UpdatedAt
	returnUser.Email = dbUser.Email

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(http.StatusCreated)
	data, err := json.Marshal(returnUser)
	if err != nil {
		rw.Write([]byte("Error kod json.Marshala za jsonMarshal za returnUser objekt. " + err.Error()))
	} else {
		rw.Write(data)
	}

}

func (cfg *apiConfig) getChirps(rw http.ResponseWriter, req *http.Request) {
	er_return := errorReturn{}

	dbChirps, err := cfg.db.GetChirps(req.Context())

	if err != nil {
		er_return.ErrorMessage = "Something went wrong. " + err.Error()
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	mappedChirps := make([]MappedChirp, len(dbChirps))
	for i, dbChirp := range dbChirps {
		mappedChirps[i] = DBChirpToMappedChirp(dbChirp)
	}

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	data, err := json.Marshal(mappedChirps)
	if err != nil {
		rw.Write([]byte("Error kod json.Marshala za jsonMarshal za returnUser objekt. " + err.Error()))
	} else {
		rw.Write(data)
	}
}

func (cfg *apiConfig) getChirpsById(rw http.ResponseWriter, req *http.Request) {
	er_return := errorReturn{}

	var param_chirpID string = req.PathValue("chirpID")
	if param_chirpID == "" {
		er_return.ErrorMessage = "No chirpID found. "
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	param_UUID, err := uuid.Parse(param_chirpID)
	if err != nil {
		er_return.ErrorMessage = "chirp convert to uuid failed with " + err.Error()
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	dbChirp, err := cfg.db.GetChirpsById(req.Context(), param_UUID)
	if err != nil {
		er_return.ErrorMessage = "Something went wrong. " + err.Error()
		returnJsonContent(rw, http.StatusNotFound, &er_return, nil) // 404
		return
	}

	mappedChirp := DBChirpToMappedChirp(dbChirp)

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	data, err := json.Marshal(mappedChirp)
	if err != nil {
		rw.Write([]byte("Error kod json.Marshala za jsonMarshal za returnUser objekt. " + err.Error()))
	} else {
		rw.Write(data)
	}

}

func (cfg *apiConfig) validateChirp(rw http.ResponseWriter, req *http.Request) {

	type parameters struct {
		BodyText string `json:"body"`
		UserID   string `json:"user_id"`
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

	chirpParams := database.CreateChirpParams{}
	chirpParams.Body = cleanBody                             // params.BodyText
	chirpParams.UserID.UUID, err = uuid.Parse(params.UserID) // If params.UserID is a string
	if err == nil {
		chirpParams.UserID.Valid = true
	} else {
		chirpParams.UserID.Valid = false
	}

	dbChirp, err := cfg.db.CreateChirp(req.Context(), chirpParams)
	if err != nil {
		er_return.ErrorMessage = "Something went wrong. CreateUser error."
		returnJsonContent(rw, http.StatusBadRequest, &er_return, nil)
		return
	}

	var returnMappedChirp MappedChirp

	returnMappedChirp.ID = dbChirp.ID
	returnMappedChirp.CreatedAt = dbChirp.CreatedAt
	returnMappedChirp.UpdatedAt = dbChirp.UpdatedAt
	returnMappedChirp.Body = dbChirp.Body
	returnMappedChirp.UserID = dbChirp.UserID.UUID.String()

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(http.StatusCreated)
	data, err := json.Marshal(returnMappedChirp)
	if err != nil {
		rw.Write([]byte("Error kod json.Marshala za jsonMarshal za returnUser objekt. " + err.Error()))
	} else {
		rw.Write(data)
	}
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	configPlatform := os.Getenv("platform")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("db error %v", err)
	}

	var dbQueries *database.Queries = database.New(db)

	// Setup dio:
	var port string = "8080"
	mux := http.NewServeMux() // *ServeMux

	var srv *http.Server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// API specification dio:
	var apiCfg apiConfig
	apiCfg.db = dbQueries
	apiCfg.platform = configPlatform

	mux.HandleFunc("/api/healthz", handleHealth)

	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpsById)
	mux.HandleFunc("GET /api/chirps", apiCfg.getChirps)

	mux.HandleFunc("POST /api/chirps", apiCfg.validateChirp)
	mux.HandleFunc("POST /api/users", apiCfg.createUser)
	mux.HandleFunc("POST /api/login", apiCfg.loginHandler)

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
