package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"

	"ripple/internal/model"
)

const StreamName = "RIPPLE_EVENTS"

type NATSQueue struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewNATSQueue(url string) (*NATSQueue, error) {
	nc, err := nats.Connect(url, nats.Timeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w", url, err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Ensure stream exists
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      StreamName,
		Subjects:  []string{"events.>"},
		Retention: nats.LimitsPolicy,
		MaxAge:    24 * time.Hour,
		Storage:   nats.FileStorage,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		// If stream configuration exists already, attempt updating it
		_, updateErr := js.UpdateStream(&nats.StreamConfig{
			Name:     StreamName,
			Subjects: []string{"events.>"},
		})
		if updateErr != nil {
			log.Printf("Warning: failed to add/update JetStream stream %s: %v", StreamName, err)
		}
	}

	return &NATSQueue{
		nc: nc,
		js: js,
	}, nil
}

func (q *NATSQueue) PublishEvent(ctx context.Context, projectID string, activity *model.Activity) error {
	data, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("failed to marshal activity: %w", err)
	}

	subject := fmt.Sprintf("events.%s.ingested", projectID)
	_, err = q.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish to JetStream subject %s: %w", subject, err)
	}
	return nil
}

func (q *NATSQueue) SubscribeEvents(durableName, queueGroup string, handler func(activity *model.Activity) error) (*nats.Subscription, error) {
	sub, err := q.js.QueueSubscribe(
		"events.>",
		queueGroup,
		func(msg *nats.Msg) {
			var act model.Activity
			if err := json.Unmarshal(msg.Data, &act); err != nil {
				log.Printf("Error: unmarshaling activity from JetStream message failed: %v", err)
				_ = msg.Ack()
				return
			}

			if err := handler(&act); err != nil {
				log.Printf("Error: handler processing activity %s failed: %v", act.EventID, err)
				// NACK message for retry if processing fails
				_ = msg.Nak()
				return
			}

			if err := msg.Ack(); err != nil {
				log.Printf("Error: acknowledging JetStream message failed: %v", err)
			}
		},
		nats.Durable(durableName),
		nats.ManualAck(),
		nats.AckWait(30*time.Second),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to JetStream queue: %w", err)
	}
	return sub, nil
}

func (q *NATSQueue) Close() {
	if q.nc != nil {
		q.nc.Close()
	}
}
