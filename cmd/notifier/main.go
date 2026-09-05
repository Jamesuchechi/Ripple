package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"

	"ripple/internal/config"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev
	},
}

func main() {
	cfg := config.Load()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Get("/v1/ws/{userID}", handleWebSocket)

	addr := fmt.Sprintf(":%s", cfg.NotifierPort)
	log.Printf("Ripple Real-time WebSocket Notifier listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Notifier server failed: %v", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket for user %s: %v", userID, err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket client connected for user: %s", userID)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket client disconnected for user: %s", userID)
			break
		}
	}
}
