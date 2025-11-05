package test

import (
	"context"
	"testing"

	"github.com/shayan/willys-mcp/internal/willys"
)

func TestCheckDeliverability(t *testing.T) {
	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	available, err := client.CheckDeliverability(context.Background(), "11151")
	if err != nil {
		t.Fatalf("Failed to check deliverability: %v", err)
	}

	t.Logf("✓ Deliverability check for 11151: %v", available)

	available2, err := client.CheckDeliverability(context.Background(), "111 51")
	if err != nil {
		t.Fatalf("Failed to check deliverability with space: %v", err)
	}

	t.Logf("✓ Deliverability check for '111 51': %v", available2)
}

func TestGetCheckoutURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	url := client.GetCheckoutURL()
	if url == "" {
		t.Error("Checkout URL is empty")
	}

	expectedURL := testBaseURL + "/kassa"
	if url != expectedURL {
		t.Errorf("Expected checkout URL %s, got %s", expectedURL, url)
	}

	t.Logf("✓ Checkout URL: %s", url)
}

func TestSetupDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	products, err := client.SearchProducts(context.Background(), "bröd", 0, 1, nil)
	if err != nil || len(products) == 0 {
		t.Skip("Could not find product to test delivery setup")
	}

	_, err = client.AddToCart(context.Background(), products[0].Code, 1)
	if err != nil {
		t.Fatalf("Failed to add product to cart: %v", err)
	}

	defer func() {
		client.ClearCart(context.Background())
	}()

	address := willys.DeliveryAddress{
		FirstName:       "Test",
		LastName:        "User",
		Address:         "Drottninggatan 1",
		PostalCode:      "11151",
		City:            "Stockholm",
		DoorCode:        "1234",
		MessageToDriver: "Test delivery",
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
		t.Fatalf("Failed to setup delivery: %v", err)
	}

	if deliveryInfo == nil {
		t.Fatal("Delivery info is nil")
	}

	if deliveryInfo.Address.FirstName != "Test" {
		t.Errorf("Expected first name 'Test', got '%s'", deliveryInfo.Address.FirstName)
	}

	t.Logf("✓ Delivery setup successful")
	t.Logf("✓ Delivery fee: %.2f kr, Picking fee: %.2f kr, Total: %.2f kr",
		deliveryInfo.DeliveryFee, deliveryInfo.PickingFee, deliveryInfo.TotalFee)
}
