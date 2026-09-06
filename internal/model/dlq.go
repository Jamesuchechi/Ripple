package model




// DLQListResponse represents the response structure for listing DLQ messages.
type DLQListResponse struct {
	Messages []DLQMessage `json:"messages"`
	Total    int          `json:"total"`
}

// DLQReplayRequest represents the request payload to replay a DLQ message.
type DLQReplayRequest struct {
	MessageID string `json:"message_id"`
}

// DLQReplayResponse represents the response structure for DLQ message replay.
type DLQReplayResponse struct {
	Status    string `json:"status"`
	MessageID string `json:"message_id"`
}
