package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jming514/chirpy/internals/jwt"
	"github.com/joho/godotenv"

	"github.com/jming514/chirpy/internals/database"

	"github.com/go-chi/chi/v5"
)

type userParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type apiConfig struct {
	DB             *database.DB
	fileserverHits int
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file", err)
		return
	}
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

	apiR.Get("/chirps", cfg.chirps)
	apiR.Get("/chirps/{chirpID}", cfg.chirp)
	apiR.Post("/chirps", cfg.createChirp)
	apiR.Delete("/chirps/{chirpID}", cfg.deleteChirp)

	apiR.Get("/users", cfg.users)
	apiR.Get("/users/{userID}", cfg.user)
	apiR.Post("/users", cfg.createUser)
	apiR.Put("/users", cfg.updateUser)

	apiR.Post("/login", cfg.login)
	apiR.Post("/refresh", cfg.refresh)
	apiR.Post("/revoke", cfg.revokeToken)

	apiR.Post("/polka/webhooks", cfg.webhooks)
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

func (cfg *apiConfig) webhooks(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("Authorization")
	strippedKey := strings.TrimPrefix(apiKey, "ApiKey ")

	goodKey := os.Getenv("API_KEY")

	if goodKey != strippedKey {
		respondWithError(w, 401, "unauthorized")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := database.UpgradeUserStruct{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Println(err)
		respondWithError(w, 500, "Error decoding parameters...")
		return
	}

	if params.Event != "user.upgraded" {
		respondWithJSON(w, 200, "ok")
		return
	}

	user, err := cfg.DB.UpgradeUser(params)
	if err != nil {
		log.Println(err)
		respondWithError(w, 500, "cannot upgrade user")
		return
	}

	respondWithJSON(w, 200, user)
}

func checkToken(r *http.Request, tokenType string) (string, error) {
	token := r.Header.Get("Authorization")
	strippedToken := strings.TrimPrefix(token, "Bearer ")

	_, err := jwt.ValidateToken(strippedToken, tokenType)
	if err != nil {
		log.Printf("Error validating token: %s\n", err)
		return "", err
	}

	return strippedToken, nil
}

func (cfg *apiConfig) revokeToken(w http.ResponseWriter, r *http.Request) {
	strippedToken, err := checkToken(r, "chirpy-refresh")
	if err != nil {
		log.Printf("Error validating token: %s\n", err)
		respondWithError(w, 500, "invalid token")
	}

	err = cfg.DB.RevokeToken(strippedToken)
	if err != nil {
		log.Printf("Error revoking token: %s\n", err)
		respondWithError(w, 500, "invalid token")
	}
}

// refresh if the current token is a refresh token and valid, return a new access token
func (cfg *apiConfig) refresh(w http.ResponseWriter, r *http.Request) {
	strippedToken, err := checkToken(r, "chirpy-refresh")
	if err != nil {
		log.Printf("Error validating token: %s\n", err)
		respondWithError(w, 500, "invalid token")
	}

	// check db if this token is revoked
	revoked, err := cfg.DB.IsTokenRevoked(strippedToken)
	if err != nil {
		log.Printf("Error checking if token is revoked: %v\n", err)
		respondWithError(w, 500, "error checking token")
		return
	}
	if revoked == true {
		respondWithJSON(w, 401, "token is revoked")
		return
	}

	respondWithJSON(w, 200, "ok")
}

func (cfg *apiConfig) login(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Error decoding parameters...")
		return
	}

	user, err := cfg.DB.Login(params.Email, params.Password)
	if err != nil {
		log.Printf("Error logging in: %s\n", err)
		respondWithError(w, 401, "Error logging in...")
		return
	}

	accessToken, err := jwt.CreateToken(60*60, user.Id, "chirpy-access")
	if err != nil {
		log.Printf("Error creating token: %s\n", err)
		respondWithError(w, 500, "error creating token...")
	}
	refreshToken, err := jwt.CreateToken(60*60*24*60, user.Id, "chirpy-refresh")
	if err != nil {
		log.Printf("Error creating token: %s\n", err)
		respondWithError(w, 500, "error creating token...")
	}

	user.Token = accessToken
	user.Refresh_Token = refreshToken

	respondWithJSON(w, 200, user)
}

func (cfg *apiConfig) user(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userID")
	theUser, err := cfg.DB.GetUser(id)
	if err != nil {
		respondWithError(w, 405, "User doesn't exist")
	}

	respondWithJSON(w, 200, theUser)
}

func (cfg *apiConfig) users(w http.ResponseWriter, r *http.Request) {
	allUsers, err := cfg.DB.GetUsers()
	if err != nil {
		respondWithError(w, 500, "Cannot get users")
	}

	respondWithJSON(w, 200, allUsers)
}

// reject if token is a refresh token
func (cfg *apiConfig) updateUser(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	strippedToken := strings.TrimPrefix(token, "Bearer ")

	validToken, err := jwt.ValidateToken(strippedToken, "chirpy-access")
	if err != nil {
		log.Printf("Error validating token: %s\n", err)
		respondWithError(w, 401, "invalid token")
		return
	}

	userId, err := jwt.GetUserIdFromToken(validToken)
	if err != nil {
		log.Printf("Error getting user ID: %s\n", err)
		respondWithError(w, 401, "cannot read user ID")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := userParams{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s\n", err)
		respondWithError(w, 500, "Error decoding parameters...")
		return
	}

	update := database.User{
		Id:       userId,
		Email:    params.Email,
		Password: params.Password,
	}

	respVals, err := cfg.DB.UpdateUser(update)
	if err != nil {
		log.Printf("Error updating user: %s\n", err)
		respondWithError(w, 500, "error updating user")
	}

	respondWithJSON(w, 200, respVals)
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := userParams{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s\n", err)
		respondWithError(w, 500, "Error decoding parameters...")
		return
	}

	// email := params.Email
	respVals, err := cfg.DB.CreateUser(params.Email, params.Password)
	if err != nil {
		log.Println(err)
		respondWithError(w, 500, "error creating user")
		return
	}

	respondWithJSON(w, 201, respVals)
}

func (cfg *apiConfig) chirp(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "chirpID")
	theChirp, err := cfg.DB.GetChirp(id)
	if err != nil {
		respondWithError(w, 404, "Chirp doesn't exist")
	}

	respondWithJSON(w, 200, theChirp)
}

func (cfg *apiConfig) chirps(w http.ResponseWriter, r *http.Request) {
	authorId := r.URL.Query().Get("author_id")

	allChirps, err := cfg.DB.GetChirps(authorId)
	if err != nil {
		log.Printf("Error getting chirps: %s\n", err)
		respondWithError(w, 500, "Cannot get chirps")
		return
	}
	respondWithJSON(w, 200, allChirps)
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	// Get user id from the token
	token := r.Header.Get("Authorization")

	strippedToken := strings.TrimPrefix(token, "Bearer ")

	validToken, err := jwt.ValidateToken(strippedToken, "chirpy-access")
	if err != nil {
		log.Printf("Error validating token: %s\n", err)
		respondWithError(w, 401, "invalid token")
		return
	}
	userId, err := jwt.GetUserIdFromToken(validToken)
	if err != nil {
		log.Printf("Error getting user ID: %s\n", err)
		respondWithError(w, 401, "cannot read user ID")
		return
	}

	type parameters struct {
		Body   string `json:"body"`
		UserId int    `json:"userId"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{
		UserId: userId,
	}

	err = decoder.Decode(&params)
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

	respVals, err := cfg.DB.CreateChirp(cleanedBody, params.UserId)
	if err != nil {
		log.Println(err)
		respondWithError(w, 500, "error creating chirp")
	}

	respondWithJSON(w, 201, respVals)
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {
	// get token
	token := r.Header.Get("Authorization")

	strippedToken := strings.TrimPrefix(token, "Bearer ")

	validToken, err := jwt.ValidateToken(strippedToken, "chirpy-access")
	if err != nil {
		log.Printf("Error validating token: %s\n", err)
		respondWithError(w, 401, "invalid token")
		return
	}

	// get user id from token
	userId, err := jwt.GetUserIdFromToken(validToken)
	if err != nil {
		log.Printf("Error getting user ID: %s\n", err)
		respondWithError(w, 401, "cannot read user ID")
		return
	}

	chirpId := chi.URLParam(r, "chirpID")
	chirpIdInt, err := strconv.Atoi(chirpId)
	if err != nil {
		log.Printf("Error converting chirp ID to int: %s\n", err)
		respondWithError(w, 500, "Invalid chirp ID")
		return
	}

	err = cfg.DB.DeleteChirp(chirpIdInt, userId)
	if err != nil {
		log.Printf("Error deleting chirp: %s\n", err)
		respondWithError(w, 403, "Cannot delete chirp")
		return
	}
	respondWithJSON(w, 200, "ok")
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

func healthz(w http.ResponseWriter, _ *http.Request) {
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
