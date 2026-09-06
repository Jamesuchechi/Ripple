package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ripple/internal/model"
)

const APIKeyPrefix = "rpl_live_"

// HashAPIKey returns the SHA-256 hex representation of a raw API key string.
func HashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}

// GenerateAPIKey creates a new raw API key and its corresponding hashed database record.
func GenerateAPIKey(projectID, name string) (string, *model.APIKey, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random bytes for API key: %w", err)
	}

	rawKey := fmt.Sprintf("%s%s", APIKeyPrefix, hex.EncodeToString(bytes))
	keyHash := HashAPIKey(rawKey)

	apiKey := &model.APIKey{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		KeyHash:   keyHash,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	return rawKey, apiKey, nil
}
