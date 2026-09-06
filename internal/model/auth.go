package model

import "time"

// APIKey represents an API key record stored in PostgreSQL.
type APIKey struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	KeyHash   string     `json:"key_hash"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// CreateAPIKeyRequest represents the payload to create a new API key.
type CreateAPIKeyRequest struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
}

// CreateAPIKeyResponse represents the response containing the raw unhashed API key (shown only once).
type CreateAPIKeyResponse struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	RawAPIKey string    `json:"raw_api_key"`
	CreatedAt time.Time `json:"created_at"`
}
