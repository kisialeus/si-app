package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	windowDuration = 60 * time.Second
	sleepDuration  = 2 * time.Second
	maxParallel    = 5
)

var (
	mu        sync.Mutex
	requests  []time.Time
	semaphore = make(chan struct{}, maxParallel)
	dataFile  string
)

// Create dir
func createDirectory(filePath string) {
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			os.Exit(1)
		}
	}
}

// Load saved requests from file
func loadRequests() {
	createDirectoryIfNeeded(dataFile)

	file, err := os.Open(dataFile)
	if err != nil {
		fmt.Println("File not found, starting with an empty request list.")
		return
	}
	defer file.Close()

	var savedRequests []time.Time
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&savedRequests); err == nil {
		requests = savedRequests
	}
}

// Save requests to file
func saveRequests() {
	createDirectory(dataFile)

	file, err := os.Create(dataFile)
	if err != nil {
		fmt.Println("Error saving requests:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.Encode(requests)
}

// Handle requests
func handler(w http.ResponseWriter, r *http.Request) {
	// Rate limit
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	time.Sleep(sleepDuration)

	mu.Lock()
	now := time.Now()

	// Cleanup data outside the 60s window
	validRequests := []time.Time{}
	for _, t := range requests {
		if now.Sub(t) <= windowDuration {
			validRequests = append(validRequests, t)
		}
	}
	requests = append(validRequests, now)
	saveRequests()
	count := len(requests)

	mu.Unlock()

	fmt.Fprintf(w, "Requests in last 60 seconds: %d\n", count)
}

// Healthz check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status": "healthy"}`)
}

func main() {
	storageDir := os.Getenv("STORAGE")
	if storageDir == "" {
		fmt.Println("ERROR: STORAGE environment variable not defined.")
		os.Exit(1)
	}

	dataFile = filepath.Join(storageDir, "requests.json")
	loadRequests()

	http.HandleFunc("/", handler)
	http.HandleFunc("/healthz", healthHandler)

	// Run the server
	fmt.Println("Server is running on :80, storing data at:", dataFile)
	http.ListenAndServe(":80", nil)
}
