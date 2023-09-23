package main

import (
	"encoding/json"
	"fmt"
	"github.com/jming514/chirpy/internals/database"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type apiConfig struct {
	fileserverHits int
	DB             *database.DB
}

func main() {
	const port = "8080"
	const filepathRoot = "."

	db, err := database.NewDB("./database.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	cfg := apiConfig{
		fileserverHits: 0,
		DB:             db,
	}
	r := chi.NewRouter()
	fsHandler := cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	r.Handle("/app", fsHandler)
	r.Handle("/app/*", fsHandler)

	apiR := chi.NewRouter()
	apiR.Get("/healthz", healthz)
	apiR.Post("/reset", cfg.reset)
	//apiR.Get("/chirps", cfg.chirps)
	apiR.Post("/chirps", cfg.createChirp)
	r.Mount("/api", apiR)

	adminR := chi.NewRouter()
	adminR.Get("/metrics", cfg.adminFsHandler)
	r.Mount("/admin", adminR)

	corsMux := middlewareCors(r)
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}

	log.Printf("Server started at %s", httpServer.Addr)
	log.Fatal(httpServer.ListenAndServe())
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s\n", err)
		respondWithError(w, 500, "Error decoding parameters...")
		return
	}

	var cleanedBody string

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	} else {
		profanity := []string{"kerfuffle", "sharbert", "fornax"}

		res := strings.Split(params.Body, " ")
		for i, v := range res {
			for _, c := range profanity {
				if strings.ToLower(v) == c {
					res[i] = "****"
				}
			}
		}
		cleanedBody = strings.Join(res, " ")
	}

	respVals, err := cfg.DB.CreateChirp(cleanedBody)
	if err != nil {
		log.Println(err)
		respondWithError(w, 500, "error ")
	}

	respondWithJSON(w, 201, respVals)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(dat)
	if err != nil {
		log.Printf("error writing: %s", err)
		return
	}
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errorResponse struct {
		Error string `json:"error"`
	}

	payload := errorResponse{
		Error: msg,
	}

	respondWithJSON(w, code, payload)
}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
}

func (cfg *apiConfig) metrics(w http.ResponseWriter, r *http.Request) {
	responseText := "Hits: " + strconv.Itoa(cfg.fileserverHits)
	_, err := w.Write([]byte(responseText))
	if err != nil {
		return
	}
}

func (cfg *apiConfig) adminFsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, err := fmt.Fprintf(w, `
<html>
<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %d times!</p>
</body>
</html>
`, cfg.fileserverHits)
	if err != nil {
		return
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		return
	}
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits += 1
		next.ServeHTTP(w, r)
	})
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
