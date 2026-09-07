package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

var store = NewStore()

func main() {
	handler := newHandler()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       5 * time.Second,
	}

	log.Printf("feature-flags listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealth)
	mux.HandleFunc("POST /flags", handleCreateFlag)
	mux.HandleFunc("GET /flags", handleListFlags)
	mux.HandleFunc("GET /flags/{key}", handleGetFlag)
	mux.HandleFunc("PUT /flags/{key}", handleUpdateFlag)
	mux.HandleFunc("DELETE /flags/{key}", handleDeleteFlag)
	mux.HandleFunc("GET /flags/{key}/evaluate", handleEvaluateFlag)
	return loggingMiddleware(recoveryMiddleware(mux))
}
