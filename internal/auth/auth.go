package auth

import (
	"encoding/json"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	passwordBytes, err := json.Marshal(password)

	if err != nil {
		return "", errors.New("failed to convert the password to bytes")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword(passwordBytes, 1)

	if err != nil {
		return "", errors.New("failed to generate a has from the password")
	}

	hashedPasswordAsString := string(hashedBytes)
	return hashedPasswordAsString, err
}

func CheckPasswordHash(password string, hash string) error {

	passwordBytes, err := json.Marshal(password)

	if err != nil {
		return errors.New("Failed to transform the password provided")
	}

	hashBytes, err := json.Marshal(hash)
	if err != nil {
		return errors.New("Please contact the administrator to reset your password")
	}

	err = bcrypt.CompareHashAndPassword(passwordBytes, hashBytes)
	return err
}
