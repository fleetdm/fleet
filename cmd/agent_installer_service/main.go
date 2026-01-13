// Package main provides the entry point for the agent installer service.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/fleetdm/fleet/v4/pkg/agent_installer_service"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := agent_installer_service.NewServer()
	log.Printf("Starting agent installer service on port %s", port)
	if err := http.ListenAndServe(":"+port, server); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
