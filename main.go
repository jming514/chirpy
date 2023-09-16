package main

import (
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

	cfg := apiConfig{}

	fsHandler := cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	r.Handle("/app", fsHandler)
	r.Handle("/app/*", fsHandler)
	r.Get("/healthz", healthz)
	r.Get("/metrics", cfg.metrics)
	r.HandleFunc("/reset", cfg.metrics)

	corsMux := middlewareCors(r)
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}
	log.Printf("Server started at %s", httpServer.Addr)
	log.Fatal(httpServer.ListenAndServe())
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
