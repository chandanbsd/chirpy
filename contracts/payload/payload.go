package payload

import "github.com/google/uuid"

type UserCreate struct {
	Email string `json:"email"`
}

type ChirpCreate struct {
	Body   string `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}
