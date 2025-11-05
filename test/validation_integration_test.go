package test

import (
	"context"
	"testing"

	"github.com/shayan/willys-mcp/internal/willys"
)

func TestInvalidProductCode(t *testing.T) {
	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.AddToCart(context.Background(), "INVALID_CODE", 1)
	if err == nil {
		t.Error("Expected error for invalid product code, got nil")
	}

	t.Logf("✓ Invalid product code rejected: %v", err)
}

func TestInvalidQuantity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.AddToCart(context.Background(), "12345_ST", 0)
	if err == nil {
		t.Error("Expected error for invalid quantity, got nil")
	}

	t.Logf("✓ Invalid quantity rejected: %v", err)

	_, err = client.AddToCart(context.Background(), "12345_ST", -1)
	if err == nil {
		t.Error("Expected error for negative quantity, got nil")
	}

	t.Logf("✓ Negative quantity rejected: %v", err)

	_, err = client.AddToCart(context.Background(), "12345_ST", 1000)
	if err == nil {
		t.Error("Expected error for excessive quantity, got nil")
	}

	t.Logf("✓ Excessive quantity rejected: %v", err)
}

func TestInvalidPostalCode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.CheckDeliverability(context.Background(), "INVALID")
	if err == nil {
		t.Error("Expected error for invalid postal code, got nil")
	}

	t.Logf("✓ Invalid postal code rejected: %v", err)
}

func TestInvalidDeliveryAddress(t *testing.T) {
	address := willys.DeliveryAddress{
		FirstName:  "",
		LastName:   "Test",
		Address:    "Test St",
		PostalCode: "12345",
		City:       "Stockholm",
	}

	err := willys.ValidateDeliveryAddress(address)
	if err == nil {
		t.Error("Expected error for missing first name, got nil")
	}

	t.Logf("✓ Invalid address rejected: %v", err)
}
