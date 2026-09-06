package model

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestIngestEventRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     IngestEventRequest
		wantErr error
	}{
		{
			name: "valid request",
			req: IngestEventRequest{
				Verb:       "post",
				ActorID:    "user_alice",
				ObjectID:   "post_123",
				Recipients: []string{"user_bob"},
				Payload:    json.RawMessage(`{"title":"Hello World"}`),
			},
			wantErr: nil,
		},
		{
			name: "missing verb",
			req: IngestEventRequest{
				ActorID:  "user_alice",
				ObjectID: "post_123",
			},
			wantErr: ErrMissingVerb,
		},
		{
			name: "missing actor_id",
			req: IngestEventRequest{
				Verb:     "post",
				ObjectID: "post_123",
			},
			wantErr: ErrMissingActorID,
		},
		{
			name: "missing object_id",
			req: IngestEventRequest{
				Verb:    "post",
				ActorID: "user_alice",
			},
			wantErr: ErrMissingObjectID,
		},
		{
			name: "invalid json payload",
			req: IngestEventRequest{
				Verb:     "post",
				ActorID:  "user_alice",
				ObjectID: "post_123",
				Payload:  json.RawMessage(`{invalid_json`),
			},
			wantErr: ErrInvalidPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
