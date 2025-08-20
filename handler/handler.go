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
	"github.com/chandanbsd/chirpy/internal/auth"
	"github.com/chandanbsd/chirpy/internal/database"
	"github.com/chandanbsd/chirpy/internal/helper"
	"github.com/google/uuid"

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

	hashedPassword, err := auth.HashPassword(payload.Password)
	if err != nil {
		resWriter.WriteHeader(500)
		resWriter.Write([]byte("Failed to create a new user"))
		return
	}

	createUserInputModel := database.CreateUserParams{
		Email:          payload.Email,
		HashedPassword: hashedPassword,
	}

	user, err := cfg.Queries.CreateUser(context.Background(), createUserInputModel)

	if err != nil {
		resWriter.WriteHeader(500)
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

		if index != len(payload.Body)-1 {
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

	createChirpInput := database.CreateChirpParams{
		Body:   payload.Body,
		UserID: payload.UserID,
	}

	res, err := cfg.Queries.CreateChirp(context.Background(), createChirpInput)

	if err != nil {
		resWriter.WriteHeader(500)
		resWriter.Write([]byte("Failed to create chirp"))
		return
	}

	dtoRes := dto.CreatedChirpDto{
		ID:        res.ID.String(),
		CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt,
		Body:      res.Body,
		UserID:    res.UserID.String(),
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

func (cfg *ApiConfig) HandleChirpsGet(resWriter http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.Queries.GetChirps(context.Background())

	if err != nil {
		resWriter.WriteHeader(500)
		resWriter.Write([]byte("Failed to fetch the chirps"))
		return
	}

	resChirps := []dto.Chirp{}

	for _, chirp := range chirps {
		newChirp := dto.Chirp{
			ID:        chirp.ID.String(),
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID.String(),
		}
		resChirps = append(resChirps, newChirp)
	}

	chirpsBytes, err := json.Marshal(resChirps)
	if err != nil {
		resWriter.WriteHeader(500)
		resWriter.Write([]byte("Internal server error"))
		return
	}

	resWriter.WriteHeader(200)
	resWriter.Write(chirpsBytes)
}

func (cfg *ApiConfig) GetChirpByChirpID(resWriter http.ResponseWriter, req *http.Request) {
	parameterName := "chirpID"

	chirpIdAsString := req.PathValue(parameterName)

	if chirpIdAsString == "" {
		resWriter.WriteHeader(404)
		resWriter.Write([]byte("Missing chirp id in the url"))
		return
	}

	chirpID, err := uuid.Parse(chirpIdAsString)

	if err != nil {
		resWriter.WriteHeader(404)
		resWriter.Write([]byte("The chirp id is invalid"))
		return
	}

	chirp, err := cfg.Queries.GetChirp(context.Background(), chirpID)
	if err != nil {
		resWriter.WriteHeader(404)
		resWriter.Write([]byte("CHirp id is not found"))
		return
	}

	resChirp := dto.Chirp{
		ID:        chirp.ID.String(),
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID.String(),
	}

	resChirpBytes, err := json.Marshal(resChirp)
	if err != nil {
		resWriter.WriteHeader(404)
		resWriter.Write([]byte("cirp may be corrupted"))
		return
	}

	resWriter.WriteHeader(200)
	resWriter.Write(resChirpBytes)
}

// func (cfg *ApiConfig) GetChirpByChirpID(resWriter http.ResponseWriter, req *http.Request) {

// }

func (cfg *ApiConfig) Login(resWriter http.ResponseWriter, req *http.Request) {

	decoder := json.NewDecoder(req.Body)
	loginPayload := payload.Login{}

	err := decoder.Decode(&loginPayload)
	if err != nil {
		helper.ReportError("Failed to deserialize the payload", resWriter, 500)
		return
	}

	userEntity, err := cfg.Queries.GetUserByEmail(context.Background(), loginPayload.Email)
	if err != nil {
		helper.ReportError("User may not exist", resWriter, 500)
		return
	}

	isAuthSuccess := auth.CheckPasswordHash(loginPayload.Password, userEntity.HashedPassword)
	if isAuthSuccess != nil {
		helper.ReportError("401 Unauthorized", resWriter, 401)
		return
	}

	userLoginSuccessDto := dto.UserLoginSuccess{
		ID:        userEntity.ID.String(),
		CreatedAt: userEntity.CreatedAt.String(),
		UpdatedAt: userEntity.UpdatedAt.String(),
		Email:     userEntity.Email,
	}

	userLoginSuccessDtoBytes, err := json.Marshal(userLoginSuccessDto)
	if err != nil {
		helper.ReportError("Unexpected failure", resWriter, 401)
		return
	}
	helper.RespondSuccess(resWriter, 200, userLoginSuccessDtoBytes)
}
