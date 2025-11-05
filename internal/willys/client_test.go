package willys

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://www.willys.se"
	username := "test@example.com"
	password := "testpassword"

	client, err := NewClient(baseURL, username, password)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client is nil")
	}

	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL %s, got %s", baseURL, client.baseURL)
	}

	if client.username != username {
		t.Errorf("Expected username %s, got %s", username, client.username)
	}

	if client.password != password {
		t.Errorf("Expected password %s, got %s", password, client.password)
	}

	if client.httpClient == nil {
		t.Error("HTTP client is nil")
	}

	if client.httpClient.Jar == nil {
		t.Error("Cookie jar is nil")
	}
}

func TestIsAuthenticated(t *testing.T) {
	client, err := NewClient("https://www.willys.se", "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.IsAuthenticated() {
		t.Error("New client should not be authenticated")
	}
}
