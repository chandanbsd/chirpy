package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func healthzHandler(resWriter http.ResponseWriter, req *http.Request) {
	header := resWriter.Header()
	header.Set("Content-Type", "text/plain; charset=utf-8")
	resWriter.WriteHeader(200)
	resWriter.Write([]byte("OK"))
	return
}

func (cfg *apiConfig) hitsHandler(resWriter http.ResponseWriter, req *http.Request) {
	header := resWriter.Header()
	header.Set("Content-Type", "text/plain; charset=utf-8")
	resWriter.WriteHeader(200)
	resWriter.Write([]byte(fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) resetHandler(resWriter http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
}

func (cfg *apiConfig) metricsHandler(resWriter http.ResponseWriter, req *http.Request) {
	header := resWriter.Header()
	header.Set("Content-Type", "text/html")
	resWriter.WriteHeader(200)
	resWriter.Write(
		[]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())),
	)
}

type ChirpPayload struct {
	Body string `json:"body"`
}

type ChirpInvalidDto struct {
	Error string `json:"error"`
}

type ChirpCleanedDto struct {
	CleanedBody string `json:"cleaned_body"`
}

func (cfg *apiConfig) validateChirpHandler(resWriter http.ResponseWriter, req *http.Request) {
	profane_words_filter := []string{"kerfuffle", "sharbert", "fornax"}

	decoder := json.NewDecoder(req.Body)
	payload := ChirpPayload{}
	header := resWriter.Header()
	header.Set("Content-Type", "application/json")
	err := decoder.Decode(&payload)

	resBody := ChirpInvalidDto{
		Error: "",
	}
	if err != nil {
		resWriter.WriteHeader(400)
		resBody.Error = "Failed to decode the request body"

		resBytes, err := json.Marshal(resBody)
		if err != nil {
			resWriter.Write(resBytes)
			return
		}
	}

	cleanedString := ""

	profaneWordFound := false
	totalWords := len(strings.Split(payload.Body, " "))
	for i, word := range strings.Split(payload.Body, " ") {
		if slices.Contains(profane_words_filter, strings.ToLower(word)) {
			cleanedString += strings.Repeat("*", 4)
			profaneWordFound = true
		} else {
			cleanedString += word
		}
		if totalWords != (i + 1) {
			cleanedString += " "
		}
	}

	if profaneWordFound == true {
		dto := ChirpCleanedDto{
			CleanedBody: cleanedString,
		}

		dtoBytes, err := json.Marshal(dto)
		if err != nil {
			panic("Failed to create response dto")
		} else {
			resWriter.WriteHeader(http.StatusOK)
			resWriter.Write(dtoBytes)
			return
		}
	}

	if len(payload.Body) <= 140 {
		resBody := ChirpCleanedDto{}
		resBody.CleanedBody = payload.Body
		resBodyBytes, err := json.Marshal(resBody)
		if err != nil {
			resBody := ChirpInvalidDto{}
			resBody.Error = "Failed to decode the request body"
			resBytes, _ := json.Marshal(resBody)
			resWriter.WriteHeader(http.StatusBadRequest)
			resWriter.Write(resBytes)
			return
		}
		resWriter.WriteHeader(http.StatusOK)
		resWriter.Write(resBodyBytes)
		return
	} else {
		resBody := ChirpInvalidDto{}
		resBody.Error = "Chirp is too long"
		resBytes, _ := json.Marshal(resBody)
		resWriter.WriteHeader(http.StatusBadRequest)
		resWriter.Write(resBytes)
		return
	}

}

func main() {
	serveMux := http.NewServeMux()

	var cfg = &apiConfig{
		fileserverHits: atomic.Int32{},
	}

	serveMux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	serveMux.HandleFunc("GET /admin/metrics/", http.HandlerFunc(cfg.metricsHandler))

	serveMux.HandleFunc("GET /api/metrics", http.HandlerFunc(cfg.hitsHandler))

	serveMux.HandleFunc("POST /api/reset", http.HandlerFunc(cfg.resetHandler))

	serveMux.HandleFunc("POST /api/validate_chirp", http.HandlerFunc(cfg.validateChirpHandler))

	server := http.Server{
		Handler: serveMux,
		Addr:    ":8080",
	}

	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
