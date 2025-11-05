package test

import (
	"context"
	"testing"

	"github.com/shayan/willys-mcp/internal/willys"
)

func TestClientCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, testUsername, testPassword)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client is nil")
	}

	t.Logf("✓ Client created successfully")
}

func TestClientAuthentication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if skipAuth {
		t.Skip("Skipping authentication test: credentials not provided (set WILLYS_USERNAME and WILLYS_PASSWORD)")
	}

	client, err := willys.NewClient(testBaseURL, testUsername, testPassword)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Log("Authenticating with headless browser...")
	err = client.LoginWithBrowser(context.Background(), testUsername, testPassword)
	if err != nil {
		t.Fatalf("Browser login failed: %v", err)
	}

	if !client.IsAuthenticated() {
		t.Error("Client should be authenticated after login")
	}

	t.Logf("✓ Browser authentication successful")

	customerInfo, err := client.GetCustomerInfo(context.Background())
	if err != nil {
		t.Fatalf("Failed to get customer info: %v", err)
	}

	if customerInfo == nil {
		t.Fatal("Customer info is nil")
	}

	if customerInfo.Email == "" {
		t.Error("Customer email is empty")
	}

	t.Logf("✓ Customer info retrieved: %s", customerInfo.Email)
}

func TestGuestMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.IsAuthenticated() {
		t.Error("Client should not be authenticated in guest mode")
	}

	t.Logf("✓ Guest mode client created successfully")
}

func TestCSRFTokenFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	token, err := client.FetchCSRFToken()
	if err != nil {
		t.Fatalf("Failed to fetch CSRF token: %v", err)
	}

	if token == "" {
		t.Error("CSRF token is empty")
	}

	t.Logf("✓ CSRF token fetched: %s", token[:20]+"...")

	token2, err := client.GetCSRFToken()
	if err != nil {
		t.Fatalf("Failed to get cached CSRF token: %v", err)
	}

	if token != token2 {
		t.Error("Cached token does not match fetched token")
	}

	t.Logf("✓ CSRF token cached correctly")
}
