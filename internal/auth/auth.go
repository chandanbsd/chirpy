package auth

import (
	"encoding/json"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func HashedPassword(password string) (string, error) {
	passwordBytes, err := json.Marshal(password)

	if err != nil {
		return "", errors.New("Failed to convert the password to bytes")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword(passwordBytes, 1000)

	if err != nil {
		return "", errors.New("Failed to generate a has from the password")
	}

	hashedPasswordAsString := string(hashedBytes)
	return hashedPasswordAsString, err
}
