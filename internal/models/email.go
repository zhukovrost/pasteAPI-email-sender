package models

type Email struct {
	To struct {
		Email string `json:"email"`
		Login string `json:"login"`
		ID    int64  `json:"id"`
	} `json:"to"`
	Type    EmailType `json:"type"`
	Message string    `json:"message"`
}

type EmailType string
