package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "thisisasecret"
	expiresIn := time.Hour * 2

	_, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf("Failed: %v", err)
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "thisisasecret"
	expiresIn := time.Hour * 2

	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf("Failed: %v", err)
	}

	actualUserID, err := ValidateJWT(tokenString, tokenSecret)
	if actualUserID != userID {
		t.Errorf("Failed: %v", err)
	}
}
