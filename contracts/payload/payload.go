package payload

import "github.com/google/uuid"

type UserCreate struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ChirpCreate struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

type Login struct {
	Password        string `json:"password"`
	Email           string `json:"email"`
	ExpiresInSecond string `json:"expires_in_seconds"`
}
