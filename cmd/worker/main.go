package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"ripple/internal/config"
	"ripple/internal/model"
	"ripple/internal/queue"
	"ripple/internal/service"
	"ripple/internal/store"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting Ripple Fanout Worker daemon (Concurrency limit: %d)...", cfg.FanoutConcurrency)

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpt)
	defer redisClient.Close()

	natsQueue, err := queue.NewNATSQueue(cfg.QueueURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS JetStream queue: %v", err)
	}
	defer natsQueue.Close()

	pgStore := store.NewPostgresStore(db)
	redisStore := store.NewRedisFeedStore(redisClient, cfg.FeedCacheSize)
	svc := service.NewEventService(pgStore, redisStore, natsQueue, cfg.CelebrityThreshold)

	// Bounded-concurrency semaphore worker pool
	sem := make(chan struct{}, cfg.FanoutConcurrency)

	sub, err := natsQueue.SubscribeEvents("ripple-fanout-worker", "fanout-workers", func(act *model.Activity) error {
		sem <- struct{}{}
		defer func() { <-sem }()

		ctx := context.Background()
		if err := svc.ProcessFanoutMessage(ctx, act); err != nil {
			log.Printf("Error processing async fanout for event %s: %v", act.EventID, err)
			return err
		}
		log.Printf("Successfully processed fanout for event %s (verb: %s, actor: %s)", act.EventID, act.Verb, act.ActorID)
		return nil
	})
	if err != nil {
		log.Fatalf("Failed subscribing to NATS JetStream events stream: %v", err)
	}
	defer sub.Unsubscribe()

	log.Printf("Ripple Fanout Worker listening on NATS JetStream subject events.> (queue group: fanout-workers)")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down worker daemon gracefully...")
}
