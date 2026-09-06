package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const defaultBaseURL = "http://localhost:8080"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "trigger":
		runTrigger(os.Args[2:])
	case "feed":
		runFeed(os.Args[2:])
	case "simulate":
		runSimulate(os.Args[2:])
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Ripple CLI Simulator v1.0.0")
	fmt.Println("Usage:")
	fmt.Println("  ripple-cli trigger   --verb <verb> --actor <actor_id> --object <object_id> [--recipient <recipient_id>] [--url <url>]")
	fmt.Println("  ripple-cli feed      --user <user_id> [--limit <n>] [--url <url>]")
	fmt.Println("  ripple-cli simulate  --rate <req_per_sec> --duration <sec> [--url <url>]")
}

func runTrigger(args []string) {
	fs := flag.NewFlagSet("trigger", flag.ExitOnError)
	verb := fs.String("verb", "post.created", "Activity verb")
	actor := fs.String("actor", "user_actor_1", "Actor ID")
	object := fs.String("object", "post_100", "Object ID")
	recipient := fs.String("recipient", "", "Optional explicit recipient ID")
	apiURL := fs.String("url", defaultBaseURL, "Ripple API base URL")
	apiKey := fs.String("apikey", "rip_live_testkey", "API Key")
	fs.Parse(args)

	payload := map[string]interface{}{
		"verb":      *verb,
		"actor_id":  *actor,
		"object_id": *object,
	}
	if *recipient != "" {
		payload["recipients"] = []string{*recipient}
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, *apiURL+"/v1/events", bytes.NewBuffer(bodyBytes))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Authorization", "Bearer "+*apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending trigger request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("[%d] Response: %s\n", resp.StatusCode, string(respBody))
}

func runFeed(args []string) {
	fs := flag.NewFlagSet("feed", flag.ExitOnError)
	user := fs.String("user", "", "User ID to read feed for")
	limit := fs.Int("limit", 20, "Feed item limit")
	apiURL := fs.String("url", defaultBaseURL, "Ripple API base URL")
	apiKey := fs.String("apikey", "rip_live_testkey", "API Key")
	fs.Parse(args)

	if *user == "" {
		fmt.Println("Error: --user is required for feed command")
		os.Exit(1)
	}

	url := fmt.Sprintf("%s/v1/feed/%s?limit=%d", *apiURL, *user, *limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Authorization", "Bearer "+*apiKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error querying feed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("[%d] Feed Output:\n%s\n", resp.StatusCode, string(respBody))
}

func runSimulate(args []string) {
	fs := flag.NewFlagSet("simulate", flag.ExitOnError)
	rate := fs.Int("rate", 50, "Requests per second target")
	duration := fs.Int("duration", 5, "Simulation duration in seconds")
	apiURL := fs.String("url", defaultBaseURL, "Ripple API base URL")
	apiKey := fs.String("apikey", "rip_live_testkey", "API Key")
	fs.Parse(args)

	fmt.Printf("Starting Ripple Load Simulator (%d req/s for %ds) against %s...\n", *rate, *duration, *apiURL)

	var successCount uint64
	var errorCount uint64

	client := &http.Client{Timeout: 3 * time.Second}
	interval := time.Second / time.Duration(*rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	stopTimer := time.After(time.Duration(*duration) * time.Second)
	var wg sync.WaitGroup

	startTime := time.Now()

Loop:
	for {
		select {
		case <-stopTimer:
			break Loop
		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				payload := map[string]interface{}{
					"verb":      "post.liked",
					"actor_id":  "sim_actor_" + fmt.Sprintf("%d", time.Now().UnixNano()%100),
					"object_id": "sim_post_999",
				}
				bodyBytes, _ := json.Marshal(payload)
				req, _ := http.NewRequest(http.MethodPost, *apiURL+"/v1/events", bytes.NewBuffer(bodyBytes))
				req.Header.Set("Authorization", "Bearer "+*apiKey)
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil || resp.StatusCode >= 400 {
					atomic.AddUint64(&errorCount, 1)
					if resp != nil {
						resp.Body.Close()
					}
					return
				}
				resp.Body.Close()
				atomic.AddUint64(&successCount, 1)
			}()
		}
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	fmt.Println("--- Simulation Results ---")
	fmt.Printf("Total Requests Sent : %d\n", successCount+errorCount)
	fmt.Printf("Successful Requests  : %d\n", successCount)
	fmt.Printf("Failed Requests      : %d\n", errorCount)
	fmt.Printf("Elapsed Duration     : %s\n", elapsed)
	fmt.Printf("Achieved Rate        : %.2f req/sec\n", float64(successCount+errorCount)/elapsed.Seconds())
}
