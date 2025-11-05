package willys

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type (
	Product struct {
		Code             string   `json:"code"`
		Name             string   `json:"name"`
		PriceValue       float64  `json:"priceValue"`
		Price            string   `json:"price"`
		ComparePrice     string   `json:"comparePrice"`
		ComparePriceUnit string   `json:"comparePriceUnit"`
		DisplayVolume    string   `json:"displayVolume"`
		Manufacturer     string   `json:"manufacturer"`
		Labels           []string `json:"labels"`
		Online           bool     `json:"online"`
		OutOfStock       bool     `json:"outOfStock"`
		SavingsAmount    *float64 `json:"savingsAmount"`
		Image            struct {
			URL string `json:"url"`
		} `json:"image"`
	}

	SearchPreferences struct {
		PriceSensitivity string   `json:"price_sensitivity"` // "cheapest" | "balanced" | "quality"
		MaxPricePerUnit  float64  `json:"max_price_per_unit"`
		RequiredLabels   []string `json:"required_labels"`
		PreferredLabels  []string `json:"preferred_labels"`
		SortBy           string   `json:"sort_by"` // "cheapest" | "best_value" | "highest_quality"
	}
)

func (c *Client) SearchProducts(ctx context.Context, query string, page, size int, prefs *SearchPreferences) ([]Product, error) {
	if query == "" {
		return nil, NewValidationError("query", "search query cannot be empty")
	}
	if page < 0 {
		return nil, NewValidationError("page", "page number cannot be negative")
	}
	if size <= 0 || size > 100 {
		return nil, NewValidationError("size", "page size must be between 1 and 100")
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("size", fmt.Sprintf("%d", size))

	searchPath := fmt.Sprintf("%s?%s", EndpointSearch, params.Encode())

	resp, err := c.DoRequest(ctx, "GET", searchPath, nil, false)
	if err != nil {
		return nil, NewAPIError(0, searchPath, "search request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewAPIError(resp.StatusCode, searchPath, "search failed", nil)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewAPIError(resp.StatusCode, searchPath, "failed to read search response", err)
	}

	var searchResponse struct {
		Results []Product `json:"results"`
	}
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		return nil, NewAPIError(resp.StatusCode, searchPath, "failed to parse search results", err)
	}

	products := searchResponse.Results

	if prefs != nil {
		products = c.filterProducts(products, prefs)
		products = c.sortProducts(products, prefs)
	}

	return products, nil
}

func (c *Client) filterProducts(products []Product, prefs *SearchPreferences) []Product {
	filtered := make([]Product, 0, len(products)/2)

	lowercaseRequired := make([]string, len(prefs.RequiredLabels))
	for i, label := range prefs.RequiredLabels {
		lowercaseRequired[i] = strings.ToLower(label)
	}

	for _, p := range products {
		if prefs.MaxPricePerUnit > 0 {
			comparePrice := parseComparePriceToFloat(p.ComparePrice)
			if comparePrice > prefs.MaxPricePerUnit {
				continue
			}
		}

		if len(lowercaseRequired) > 0 {
			productLabelsLower := make([]string, len(p.Labels))
			for i, label := range p.Labels {
				productLabelsLower[i] = strings.ToLower(label)
			}

			hasAllRequired := true
			for _, reqLabel := range lowercaseRequired {
				found := false
				for _, label := range productLabelsLower {
					if strings.Contains(label, reqLabel) {
						found = true
						break
					}
				}
				if !found {
					hasAllRequired = false
					break
				}
			}
			if !hasAllRequired {
				continue
			}
		}

		filtered = append(filtered, p)
	}

	return filtered
}

func parseComparePriceToFloat(priceStr string) float64 {
	priceStr = strings.TrimSuffix(priceStr, " kr")
	priceStr = strings.ReplaceAll(priceStr, ",", ".")
	price, _ := strconv.ParseFloat(priceStr, 64)
	return price
}

func (c *Client) sortProducts(products []Product, prefs *SearchPreferences) []Product {
	sort.Slice(products, func(i, j int) bool {
		pi, pj := products[i], products[j]

		switch prefs.SortBy {
		case "cheapest":
			iPrice := parseComparePriceToFloat(pi.ComparePrice)
			jPrice := parseComparePriceToFloat(pj.ComparePrice)
			return iPrice < jPrice

		case "best_value":

			iScore := c.calculateValueScore(pi)
			jScore := c.calculateValueScore(pj)
			return iScore > jScore

		case "highest_quality":
			iLabels := len(pi.Labels)
			jLabels := len(pj.Labels)
			if iLabels != jLabels {
				return iLabels > jLabels
			}
			iPrice := parseComparePriceToFloat(pi.ComparePrice)
			jPrice := parseComparePriceToFloat(pj.ComparePrice)
			return iPrice < jPrice

		default:

			return false
		}
	})

	return products
}

func (c *Client) calculateValueScore(p Product) float64 {
	score := 0.0

	comparePrice := parseComparePriceToFloat(p.ComparePrice)
	if comparePrice > 0 {
		score += 100.0 / comparePrice
	}

	qualityLabels := []string{"krav", "ekologisk", "nyckelhål", "svensk"}
	for _, label := range p.Labels {
		labelLower := strings.ToLower(label)
		for _, quality := range qualityLabels {
			if strings.Contains(labelLower, quality) {
				score += 10.0
				break
			}
		}
	}

	if p.SavingsAmount != nil && *p.SavingsAmount > 0 {
		score += *p.SavingsAmount * 0.5
	}

	return score
}
