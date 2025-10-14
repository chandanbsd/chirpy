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
	Password         string `json:"password"`
	Email            string `json:"email"`
}

type UpdateCredentialsPayload struct {
	Password         string `json:"password"`
	Email            string `json:"email"`
}

// {
//   "event": "user.upgraded",
//   "data": {
//     "user_id": "3311741c-680c-4546-99f3-fc9efac2036c"
//   }
// }

type UpgradeUserToChirpRedWebhook struct {
	Event	string `json:"event"`
	Data struct {
		UserID string `json:"user_id"`
	} `json:"data"`

}
