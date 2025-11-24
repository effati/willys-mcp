package test

import (
	"context"
	"testing"

	"github.com/effati/willys-mcp/internal/willys"
)

func TestAddToCart(t *testing.T) {
	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	products, err := client.SearchProducts(context.Background(), "mjölk", 0, 1, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(products) == 0 {
		t.Skip("No products found to add to cart")
	}

	productCode := products[0].Code
	t.Logf("Adding product to cart: %s (%s)", products[0].Name, productCode)

	cart, err := client.AddToCart(context.Background(), productCode, 2)
	if err != nil {
		t.Fatalf("Failed to add to cart: %v", err)
	}

	if cart == nil {
		t.Fatal("Cart is nil")
	}

	found := false
	for _, item := range cart.Items {
		if item.ProductCode == productCode {
			found = true
			if item.Quantity != 2 {
				t.Errorf("Expected quantity 2, got %d", item.Quantity)
			}
		}
	}

	if !found {
		t.Error("Product not found in cart after adding")
	}

	t.Logf("✓ Product added to cart successfully")
	t.Logf("✓ Cart total: %.2f kr (%d items)", cart.TotalPrice, cart.ItemCount)

	_, err = client.RemoveFromCart(context.Background(), productCode, 0)
	if err != nil {
		t.Logf("Warning: Failed to cleanup cart: %v", err)
	}
}

func TestViewCart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	cart, err := client.GetCart(context.Background())
	if err != nil {
		t.Fatalf("Failed to get cart: %v", err)
	}

	if cart == nil {
		t.Fatal("Cart is nil")
	}

	t.Logf("✓ Cart retrieved successfully")
	t.Logf("✓ Cart has %d items, total: %.2f kr", cart.ItemCount, cart.TotalPrice)
}

func TestRemoveFromCart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	products, err := client.SearchProducts(context.Background(), "mjölk", 0, 1, nil)
	if err != nil || len(products) == 0 {
		t.Skip("Could not find product to test removal")
	}

	productCode := products[0].Code

	_, err = client.AddToCart(context.Background(), productCode, 3)
	if err != nil {
		t.Fatalf("Failed to add to cart: %v", err)
	}

	t.Logf("Added 3 of product %s", productCode)

	cart, err := client.RemoveFromCart(context.Background(), productCode, 1)
	if err != nil {
		t.Fatalf("Failed to remove from cart: %v", err)
	}

	found := false
	for _, item := range cart.Items {
		if item.ProductCode == productCode {
			found = true
			if item.Quantity != 2 {
				t.Errorf("Expected quantity 2 after removal, got %d", item.Quantity)
			}
		}
	}

	if !found {
		t.Error("Product not found in cart after partial removal")
	}

	t.Logf("✓ Removed 1 item, 2 remaining")

	cart, err = client.RemoveFromCart(context.Background(), productCode, 0)
	if err != nil {
		t.Fatalf("Failed to remove all from cart: %v", err)
	}

	for _, item := range cart.Items {
		if item.ProductCode == productCode {
			t.Errorf("Product still in cart after removing all")
		}
	}

	t.Logf("✓ Removed all items successfully")
}

func TestClearCart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	products, err := client.SearchProducts(context.Background(), "vatten", 0, 1, nil)
	if err == nil && len(products) > 0 {
		client.AddToCart(context.Background(), products[0].Code, 1)
	}

	err = client.ClearCart(context.Background())
	if err != nil {
		t.Fatalf("Failed to clear cart: %v", err)
	}

	t.Logf("✓ Cart cleared")

	cart, err := client.GetCart(context.Background())
	if err != nil {
		t.Fatalf("Failed to get cart after clearing: %v", err)
	}

	if cart.ItemCount != 0 {
		t.Errorf("Cart should be empty, but has %d items", cart.ItemCount)
	}

	t.Logf("✓ Cart verified empty")
}
