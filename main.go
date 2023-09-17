package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"strconv"
)

type apiConfig struct {
	fileserverHits int
}

func main() {
	port := "8080"
	r := chi.NewRouter()
	apiR := chi.NewRouter()
	adminR := chi.NewRouter()
	r.Mount("/api", apiR)
	r.Mount("/admin", adminR)

	cfg := apiConfig{}

	fsHandler := cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	r.Handle("/app", fsHandler)
	r.Handle("/app/*", fsHandler)
	apiR.Get("/healthz", healthz)
	apiR.HandleFunc("/reset", cfg.reset)
	apiR.Post("/validate_chirp", cfg.validateChirp)

	adminR.Get("/metrics", cfg.adminFsHandler)

	corsMux := middlewareCors(r)
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}

	log.Printf("Server started at %s", httpServer.Addr)
	log.Fatal(httpServer.ListenAndServe())
}

func (cfg *apiConfig) validateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		log.Printf("Error decoding paramters: %s", err)
		w.WriteHeader(500)
		return
	}

	fmt.Printf(strconv.Itoa(len(params.Body)))

	type returnVals struct {
		Valid bool   `json:"valid"`
		Error string `json:"error"`
	}

	respVals := returnVals{}

	if len(params.Body) > 140 {
		w.WriteHeader(400)
		respVals.Error = "Chirp is too long"
	} else {
		respVals.Valid = true
	}

	dat, err := json.Marshal(respVals)

	fmt.Printf(string(dat))

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, err = w.Write(dat)
	if err != nil {
		log.Printf("error writing: %s", err)
		return
	}
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
	_, err := w.Write([]byte(fmt.Sprintf(`
<html>
<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %d times!</p>
</body>
</html>
`, cfg.fileserverHits)))
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
