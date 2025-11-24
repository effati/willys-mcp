package test

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/effati/willys-mcp/internal/willys"
	"github.com/joho/godotenv"
)

func init() {
	envPath := filepath.Join("..", ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("No .env file found or error loading it: %v (this is okay if using environment variables)", err)
	}
}

var (
	testBaseURL  = getEnv("WILLYS_BASE_URL", "https://www.willys.se")
	testUsername = os.Getenv("WILLYS_USERNAME")
	testPassword = os.Getenv("WILLYS_PASSWORD")
	skipAuth     = testUsername == "" || testPassword == ""
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestCompleteShoppingWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Log("Starting complete shopping workflow...")

	t.Log("Step 1: Searching for milk...")
	milkProducts, err := client.SearchProducts(context.Background(), "mjölk", 0, 5, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(milkProducts) == 0 {
		t.Skip("No milk products found")
	}
	t.Logf("✓ Found %d milk products", len(milkProducts))

	t.Log("Step 2: Adding product to cart...")
	cart, err := client.AddToCart(context.Background(), milkProducts[0].Code, 2)
	if err != nil {
		t.Fatalf("Add to cart failed: %v", err)
	}
	t.Logf("✓ Cart total: %.2f kr (%d items)", cart.TotalPrice, cart.ItemCount)

	t.Log("Step 3: Searching for bread...")
	breadProducts, err := client.SearchProducts(context.Background(), "bröd", 0, 5, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(breadProducts) > 0 {
		t.Log("Step 4: Adding bread to cart...")
		cart, err = client.AddToCart(context.Background(), breadProducts[0].Code, 1)
		if err != nil {
			t.Fatalf("Add to cart failed: %v", err)
		}
		t.Logf("✓ Cart total: %.2f kr (%d items)", cart.TotalPrice, cart.ItemCount)
	}

	t.Log("Step 5: Viewing cart...")
	cart, err = client.GetCart(context.Background())
	if err != nil {
		t.Fatalf("Get cart failed: %v", err)
	}
	t.Logf("✓ Cart contains %d items:", cart.ItemCount)
	for _, item := range cart.Items {
		t.Logf("  - %s x%d = %.2f kr", item.Name, item.Quantity, item.TotalPrice)
	}

	t.Log("Step 6: Setting up delivery...")
	address := willys.DeliveryAddress{
		FirstName:  "Test",
		LastName:   "User",
		Address:    "Drottninggatan 1",
		PostalCode: "11151",
		City:       "Stockholm",
	}

	slots, err := client.GetAvailableTimeSlots(context.Background(), address.PostalCode)
	if err != nil {
		t.Fatalf("Failed to get time slots: %v", err)
	}

	if len(slots) == 0 {
		t.Skip("No available delivery slots for this postal code")
	}

	slot := slots[0]
	t.Logf("Using slot: %s %s-%s", slot.Date, slot.StartTime, slot.EndTime)

	deliveryInfo, err := client.SetupDelivery(context.Background(), address, slot)
	if err != nil {
		t.Fatalf("Setup delivery failed: %v", err)
	}
	t.Logf("✓ Delivery scheduled for %s between %s-%s",
		deliveryInfo.TimeSlot.Date,
		deliveryInfo.TimeSlot.StartTime,
		deliveryInfo.TimeSlot.EndTime)

	t.Log("Step 7: Getting checkout URL...")
	checkoutURL := client.GetCheckoutURL()
	t.Logf("✓ Checkout URL: %s", checkoutURL)

	t.Log("Step 8: Cleaning up cart...")
	err = client.ClearCart(context.Background())
	if err != nil {
		t.Fatalf("Clear cart failed: %v", err)
	}
	t.Log("✓ Cart cleared")

	t.Log("✅ Complete shopping workflow finished successfully!")
}

func TestMultipleItemsWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	defer client.ClearCart(context.Background())

	t.Log("Testing workflow with multiple items...")

	searchQueries := []string{"mjölk", "bröd", "smör", "ost", "ägg"}

	totalItems := 0

	for _, query := range searchQueries {
		t.Logf("Searching for: %s", query)
		products, err := client.SearchProducts(context.Background(), query, 0, 1, nil)
		if err != nil {
			t.Logf("Warning: Search for '%s' failed: %v", query, err)
			continue
		}

		if len(products) == 0 {
			t.Logf("Warning: No products found for '%s'", query)
			continue
		}

		cart, err := client.AddToCart(context.Background(), products[0].Code, 1)
		if err != nil {
			t.Logf("Warning: Failed to add '%s' to cart: %v", query, err)
			continue
		}

		totalItems++
		t.Logf("✓ Added %s to cart. Cart total: %.2f kr", products[0].Name, cart.TotalPrice)
	}

	cart, err := client.GetCart(context.Background())
	if err != nil {
		t.Fatalf("Failed to get final cart: %v", err)
	}

	t.Logf("✓ Final cart has %d different items", len(cart.Items))
	t.Logf("✓ Total: %.2f kr", cart.FinalTotal)

	if totalItems > 0 {
		t.Logf("✅ Successfully added %d items to cart", totalItems)
	}
}
