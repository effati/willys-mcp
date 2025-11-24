package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/effati/willys-mcp/internal/willys"
)

func parseComparePrice(priceStr string) float64 {
	var price float64
	priceStr = trimSuffix(priceStr, " kr")
	priceStr = replaceAll(priceStr, ",", ".")
	fmt.Sscanf(priceStr, "%f", &price)
	return price
}

func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i <= len(s)-len(old) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}
func TestBasicProductSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	products, err := client.SearchProducts(context.Background(), "mjölk", 0, 10, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(products) == 0 {
		t.Error("No products found for 'mjölk'")
	}

	t.Logf("✓ Found %d products for 'mjölk'", len(products))

	if len(products) > 0 {
		p := products[0]
		if p.Code == "" {
			t.Error("Product code is empty")
		}
		if p.Name == "" {
			t.Error("Product name is empty")
		}
		if p.PriceValue <= 0 {
			t.Error("Product price is invalid")
		}

		t.Logf("✓ First product: %s (Code: %s, Price: %.2f kr)", p.Name, p.Code, p.PriceValue)
	}
}

func TestSearchWithPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	page0, err := client.SearchProducts(context.Background(), "bröd", 0, 5, nil)
	if err != nil {
		t.Fatalf("Search page 0 failed: %v", err)
	}

	page1, err := client.SearchProducts(context.Background(), "bröd", 1, 5, nil)
	if err != nil {
		t.Fatalf("Search page 1 failed: %v", err)
	}

	if len(page0) == 0 {
		t.Error("Page 0 has no results")
	}

	t.Logf("✓ Page 0: %d products, Page 1: %d products", len(page0), len(page1))

	if len(page0) > 0 && len(page1) > 0 {
		if page0[0].Code == page1[0].Code {
			t.Error("Page 0 and page 1 have the same first product")
		}
	}
}

func TestSearchWithFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	prefs := &willys.SearchPreferences{
		MaxPricePerUnit: 50.0, // Max 50 kr per kg/l
	}

	products, err := client.SearchProducts(context.Background(), "mjölk", 0, 20, prefs)
	if err != nil {
		t.Fatalf("Search with filtering failed: %v", err)
	}

	t.Logf("✓ Found %d products with price filter", len(products))

	for _, p := range products {
		comparePrice := parseComparePrice(p.ComparePrice)
		if comparePrice > 50.0 {
			t.Errorf("Product %s exceeds price limit: %.2f kr/unit", p.Name, comparePrice)
		}
	}

	if len(products) > 0 {
		t.Logf("✓ All products within price limit")
	}
}

func TestSearchWithLabelFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	prefs := &willys.SearchPreferences{
		RequiredLabels: []string{"Ekologisk"},
	}

	products, err := client.SearchProducts(context.Background(), "mjölk", 0, 20, prefs)
	if err != nil {
		t.Fatalf("Search with label filtering failed: %v", err)
	}

	t.Logf("✓ Found %d organic products", len(products))

	for _, p := range products {
		hasLabel := false
		for _, label := range p.Labels {
			if label == "Ekologisk" {
				hasLabel = true
				break
			}
		}
		if !hasLabel {
			t.Errorf("Product %s missing required label: Ekologisk (has: %v)", p.Name, p.Labels)
		}
	}

	if len(products) > 0 {
		t.Logf("✓ All products have required label")
	}
}

func TestSearchWithSorting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := willys.NewClient(testBaseURL, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	prefs := &willys.SearchPreferences{
		SortBy: "cheapest",
	}

	products, err := client.SearchProducts(context.Background(), "vatten", 0, 10, prefs)
	if err != nil {
		t.Fatalf("Search with sorting failed: %v", err)
	}

	if len(products) > 1 {
		for i := 1; i < len(products); i++ {
			iPrice := parseComparePrice(products[i].ComparePrice)
			iPrevPrice := parseComparePrice(products[i-1].ComparePrice)
			if iPrice < iPrevPrice {
				t.Errorf("Products not sorted by price: product %d (%.2f) < product %d (%.2f)",
					i, iPrice, i-1, iPrevPrice)
			}
		}
		t.Logf("✓ Products sorted by price correctly")
	}
}
