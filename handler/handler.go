package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/chandanbsd/chirpy/contracts/dto"
	"github.com/chandanbsd/chirpy/contracts/payload"
	"github.com/chandanbsd/chirpy/internal/database"

	_ "github.com/lib/pq"
)

type ApiConfig struct {
	FileserverHits atomic.Int32
	Queries        *database.Queries
	Platform       string
}

func HealthzHandler(resWriter http.ResponseWriter, req *http.Request) {
	header := resWriter.Header()
	header.Set("Content-Type", "text/plain; charset=utf-8")
	resWriter.WriteHeader(200)
	resWriter.Write([]byte("OK"))
	return
}

func (cfg *ApiConfig) HitsHandler(resWriter http.ResponseWriter, req *http.Request) {
	header := resWriter.Header()
	header.Set("Content-Type", "text/plain; charset=utf-8")
	resWriter.WriteHeader(200)
	resWriter.Write([]byte(fmt.Sprintf("Hits: %v", cfg.FileserverHits.Load())))
}

func (cfg *ApiConfig) ResetHandler(resWriter http.ResponseWriter, req *http.Request) {
	cfg.FileserverHits.Store(0)

	if cfg.Platform != "dev" {
		resWriter.WriteHeader(403)
		resWriter.Write([]byte("Not authorized to perform this action on the given environment"))
		return
	}

	err := cfg.Queries.DeleteUsers(context.Background())
	if err != nil {
		resWriter.WriteHeader(500)
		resWriter.Write([]byte("Failed to delete the users"))
		return
	}

	resWriter.WriteHeader(200)
	resWriter.Write([]byte("Users deleted"))
}

func (cfg *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *ApiConfig) MetricsHandler(resWriter http.ResponseWriter, req *http.Request) {
	header := resWriter.Header()
	header.Set("Context-Type", "text/html")
	resWriter.WriteHeader(200)
	resWriter.Write([]byte(fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
`, cfg.FileserverHits.Load())))
}

func (cfg *ApiConfig) HandleValidateChirp(resWriter http.ResponseWriter, req *http.Request) {

	type payload struct {
		body string `json:"body"`
	}

	type dto struct {
		error string `json:"error"`
	}

	type dtoValid struct {
		body string `json:"body"`
	}

	defer req.Body.Close()
	decode := json.NewDecoder(req.Body)

	var p payload

	err := decode.Decode(&p)
	if err != nil {
		d := dtoValid{
			body: "This is an opinion I need to share with the world",
		}
		dataBytes, _ := json.Marshal(d)

		resWriter.WriteHeader(400)
		resWriter.Write(dataBytes)
	} else {
		d := dto{
			error: "Something went wrong",
		}
		dataBytes, _ := json.Marshal(d)
		resWriter.WriteHeader(200)
		resWriter.Write(dataBytes)
	}
}

func (cfg *ApiConfig) HandleUserCreation(resWriter http.ResponseWriter, req *http.Request) {
	payload := payload.UserCreate{}

	defer req.Body.Close()

	decoder := json.NewDecoder(req.Body)

	err := decoder.Decode(&payload)
	if err != nil || payload.Email == "" {
		resWriter.WriteHeader(400)
		resWriter.Write([]byte("Invalid payload"))
		return
	}

	user, err := cfg.Queries.CreateUser(context.Background(), payload.Email)

	if err != nil {
		resWriter.WriteHeader(400)
		resWriter.Write([]byte("Failed to insert the user into the database"))
		return
	}

	userDto := dto.UserDto{
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		ID:        user.ID.String(),
	}

	resWriter.WriteHeader(201)
	dataBytes, _ := json.Marshal(userDto)
	resWriter.Write(dataBytes)
}

func (cfg *ApiConfig) HandleChirpCreate(resWriter http.ResponseWriter, req *http.Request) {
	payload := payload.ChirpCreate{}
	defer req.Body.Close()

	decoder := json.NewDecoder(req.Body)

	badWords := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}

	badWordsMap := make(map[string]bool)

	for _, word := range badWords {
		badWordsMap[word] = true
	}

	err := decoder.Decode(&payload)
	if err != nil || payload.Body == "" {
		resWriter.WriteHeader(400)
		resWriter.Write([]byte("Invalid payload"))
		return
	}

	cleanedChirpDto := dto.CleanedChirpDto{}
	containsProfanity := false

	cleanedChirp := ""

	for index, word := range strings.Split(payload.Body, " ") {
		if badWordsMap[strings.ToLower(word)] == true {
			containsProfanity = true
			cleanedChirp += "****"
		} else {
			cleanedChirp = cleanedChirp + " " + word
		}

		if index != len(payload.Body) - 1 {
			cleanedChirp += " "
		}	
	}

	cleanedChirpDto.CleanedChirp = cleanedChirp
	cleanedChirpBytes, err := json.Marshal(cleanedChirpDto)

	if containsProfanity && err != nil {
		resWriter.WriteHeader(400)
		resWriter.Write(cleanedChirpBytes)
		return
	}

	createChirpInput := database.CreateChirpParams {
		Body:  payload.Body,
		UserID: payload.UserID,
	}

	res, err := cfg.Queries.CreateChirp(context.Background(), createChirpInput)


	if err != nil {
		resWriter.WriteHeader(500)
		resWriter.Write([]byte("Failed to create chirp"))
		return
	}

	dtoRes := dto.CreatedChirpDto {
		ID: res.ID.String(),
		CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt,
		Body: res.Body,
		UserID: res.UserID.String(),
	}

	resBodyBytes, err := json.Marshal(dtoRes)

	if err != nil {
		resWriter.WriteHeader(500)
		resWriter.Write([]byte("Failed to parse json"))
		return
	}


	resWriter.WriteHeader(201)
	resWriter.Write(resBodyBytes)
}
