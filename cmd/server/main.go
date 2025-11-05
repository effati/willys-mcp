package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/shayan/willys-mcp/internal/willys"
	"github.com/shayan/willys-mcp/pkg/mcp"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading it: %v", err)
	}

	baseURL := os.Getenv("WILLYS_BASE_URL")
	if baseURL == "" {
		baseURL = "https://www.willys.se"
	}

	username := os.Getenv("WILLYS_USERNAME")
	if username == "" {
		log.Fatalf("WILLYS_USERNAME environment variable is required")
	}

	password := os.Getenv("WILLYS_PASSWORD")
	if password == "" {
		log.Fatalf("WILLYS_PASSWORD environment variable is required")
	}

	client, err := willys.NewClient(baseURL, username, password)
	if err != nil {
		log.Fatalf("Failed to create Willys client: %v", err)
	}

	log.Println("Authenticating with Willys (using headless browser)...")
	if err := client.LoginWithBrowser(context.Background(), username, password); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	log.Println("Successfully authenticated")

	server := mcp.NewServer(client)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
