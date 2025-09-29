package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

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
	JWTSecret      string
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

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		helper.ReportError("The request does not contain the bearer token", resWriter, 401)
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.JWTSecret)
	if err != nil {
		helper.ReportError("The token validation has failurd", resWriter, 401)
		return
	}

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

	err = decoder.Decode(&payload)
	if err != nil || payload.Body == "" {
		resWriter.WriteHeader(400)
		resWriter.Write([]byte("Invalid payload"))
		return
	}

	cleanedChirpDto := dto.CleanedChirpDto{}
	containsProfanity := false
    cleanedWords := []string{}


    for _, word := range strings.Fields(payload.Body) {
        if badWordsMap[strings.ToLower(word)] {
            containsProfanity = true
            cleanedWords = append(cleanedWords, "****")
        } else {
            cleanedWords = append(cleanedWords, word)
        }
    }

	cleanedChirpDto.CleanedChirp = strings.Join(cleanedWords, " ")
	cleanedChirpBytes, err := json.Marshal(cleanedChirpDto)

	if containsProfanity {
		resWriter.WriteHeader(400)
		resWriter.Write(cleanedChirpBytes)
		return
	}

	createChirpInput := database.CreateChirpParams{
		Body:   payload.Body,
		UserID: userId,
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

	var expirationDuration time.Duration = time.Hour

	token, err := auth.MakeJWT(
		userEntity.ID,
		cfg.JWTSecret,
		expirationDuration,
	)

	if err != nil {
		helper.ReportError("Failed to generate a token for the user", resWriter, 500)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		return
	}

	createRefreshTokenParams := database.CreateRefreshTokenParams {
		Token: refreshToken,
		UserID: userEntity.ID,
		ExpiresAt: time.Now().AddDate(0, 1, 0),
		RevokedAt: sql.NullTime{},
	}

	_, err = cfg.Queries.CreateRefreshToken(context.Background(), createRefreshTokenParams)
	if err != nil {
		helper.ReportError("Failed to create refresh token", resWriter, 500)
		return
	}

	userLoginSuccessDto := dto.UserLoginSuccess{
		ID:        userEntity.ID.String(),
		CreatedAt: userEntity.CreatedAt.String(),
		UpdatedAt: userEntity.UpdatedAt.String(),
		Email:     userEntity.Email,
		Token:     token,
		RefreshToken: refreshToken,
	}

	userLoginSuccessDtoBytes, err := json.Marshal(userLoginSuccessDto)
	if err != nil {
		helper.ReportError("Unexpected failure", resWriter, 401)
		return
	}
	helper.RespondSuccess(resWriter, 200, userLoginSuccessDtoBytes)
}

func (cfg *ApiConfig) Refresh(resWriter http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		helper.ReportError("Unable to refresh", resWriter, 500)
		return
	}

	tokenFromDB, err := cfg.Queries.GetRefreshToken(context.Background(), token)
		if err != nil {
		helper.ReportError("Refresh token was not generated by the system", resWriter, 500)
		return
	}

	tokenFromRefresh := dto.TokenFromRefresh{}

	if tokenFromDB.ExpiresAt.After(time.Now()) && !tokenFromDB.RevokedAt.Valid {
		newToken, err := auth.MakeJWT(tokenFromDB.UserID, cfg.JWTSecret, time.Hour)
		if err != nil {
			helper.ReportError("Failed to generate a new token", resWriter, 500)
			return
		}

		tokenFromRefresh.Token = newToken

		resBytes, err := json.Marshal(tokenFromRefresh)
				if err != nil {
			helper.ReportError("Failed to generate body bytes", resWriter, 500)
			return
		}

		helper.RespondSuccess(resWriter, 200, resBytes)
	} else {
		helper.ReportError("Refresh token has expired authenticate again", resWriter, 401)
	}
}

func (cfg *ApiConfig) Revoke(resWriter http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		helper.ReportError("Unable to revoke as no token found", resWriter, 500)
		return
	}

	_, err = cfg.Queries.GetRefreshToken(context.Background(), token)
	if err != nil {
		helper.ReportError("Unable to revoke as no token found", resWriter, 500)
		return
	}

	err  = cfg.Queries.RevokeToken(context.Background(), token)
	if err != nil {
		helper.ReportError("Failed to revoke token", resWriter, 500)
		return
	}

	helper.RespondSuccess(resWriter, 204, []byte{})
}

func (cfg *ApiConfig) UpdateCredential(resWriter http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		helper.ReportError("Invalid authentication token", resWriter, 401)
		return
	}

	userGuid, err := auth.ValidateJWT(token, cfg.JWTSecret)
	if err != nil {
		helper.ReportError("Invalid authentication token", resWriter, 401)
		return
	}

	decoder := json.NewDecoder(req.Body)
	payload := payload.UpdateCredentialsPayload{}
	err = decoder.Decode(&payload)
	if err != nil {
		helper.ReportError("Failed to decode the request body", resWriter, 401)
		return
	}

	hashedPassword, err := auth.HashPassword(payload.Password)
	if err != nil {
		helper.ReportError("Failed to generate the password hash", resWriter, 500)
		return
	}

	userCredentialsParam := database.UpdateUserCredentialParams{
		ID: userGuid,
		Email: payload.Email,
		HashedPassword: hashedPassword,
	}

	payload.Password = hashedPassword
	err = cfg.Queries.UpdateUserCredential(context.Background(), userCredentialsParam)
	if err != nil {
		helper.ReportError("Failed to update credentials", resWriter, 500)
		return
	}

	user, err := cfg.Queries.GetUserById(context.Background(), userGuid)
	if err != nil {
		helper.ReportError("Failed to generate the dto", resWriter, 500)
	}
	dto := dto.UserDto{
		Email: user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		ID: user.ID.String(),
	}

	dtoBytes, err := json.Marshal(dto)

	helper.RespondSuccess(resWriter, 200, dtoBytes)
}
