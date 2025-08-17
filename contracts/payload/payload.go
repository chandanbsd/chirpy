package payload

type UserCreate struct {
	Email string `json:"email"`
}

type ChirpCreate struct {
	Body   string `json:"body"`
	UserID string `json:"user_id"`
}
