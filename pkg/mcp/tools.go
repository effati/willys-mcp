package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/shayan/willys-mcp/internal/willys"
)

type ToolHandler struct {
	client willys.WillysAPI
}

func NewToolHandler(client willys.WillysAPI) *ToolHandler {
	return &ToolHandler{client: client}
}

func (h *ToolHandler) SearchGroceries(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := mcp.ParseString(request, "query", "")
	if query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	page := mcp.ParseInt(request, "page", 0)
	size := mcp.ParseInt(request, "size", 30)

	var prefs *willys.SearchPreferences
	if prefsData := mcp.ParseStringMap(request, "preferences", nil); prefsData != nil {
		prefs = &willys.SearchPreferences{}
		if ps, ok := prefsData["price_sensitivity"].(string); ok {
			prefs.PriceSensitivity = ps
		}
		if mpu, ok := prefsData["max_price_per_unit"].(float64); ok {
			prefs.MaxPricePerUnit = mpu
		}
		if rl, ok := prefsData["required_labels"].([]any); ok {
			for _, label := range rl {
				if l, ok := label.(string); ok {
					prefs.RequiredLabels = append(prefs.RequiredLabels, l)
				}
			}
		}
		if pl, ok := prefsData["preferred_labels"].([]any); ok {
			for _, label := range pl {
				if l, ok := label.(string); ok {
					prefs.PreferredLabels = append(prefs.PreferredLabels, l)
				}
			}
		}
		if sb, ok := prefsData["sort_by"].(string); ok {
			prefs.SortBy = sb
		}
	}

	products, err := h.client.SearchProducts(ctx, query, page, size, prefs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]any{
		"products": products,
		"count":    len(products),
	})
}

func (h *ToolHandler) AddToCart(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	productCode := mcp.ParseString(request, "product_code", "")
	if productCode == "" {
		return mcp.NewToolResultError("product_code parameter is required"), nil
	}

	quantity := mcp.ParseInt(request, "quantity", 1)

	cart, err := h.client.AddToCart(ctx, productCode, quantity)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add to cart: %v", err)), nil
	}

	return mcp.NewToolResultJSON(cart)
}

func (h *ToolHandler) ViewCart(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cart, err := h.client.GetCart(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get cart: %v", err)), nil
	}

	return mcp.NewToolResultJSON(cart)
}

func (h *ToolHandler) RemoveFromCart(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	productCode := mcp.ParseString(request, "product_code", "")
	if productCode == "" {
		return mcp.NewToolResultError("product_code parameter is required"), nil
	}

	quantity := mcp.ParseInt(request, "quantity", 0)

	cart, err := h.client.RemoveFromCart(ctx, productCode, quantity)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove from cart: %v", err)), nil
	}

	return mcp.NewToolResultJSON(cart)
}

func (h *ToolHandler) SelectDeliveryTime(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	addressData := mcp.ParseStringMap(request, "address", nil)
	if addressData == nil {
		return mcp.NewToolResultError("address parameter is required"), nil
	}

	address := willys.DeliveryAddress{
		FirstName:       getStringField(addressData, "first_name"),
		LastName:        getStringField(addressData, "last_name"),
		Address:         getStringField(addressData, "address"),
		PostalCode:      getStringField(addressData, "postal_code"),
		City:            getStringField(addressData, "city"),
		DoorCode:        getStringField(addressData, "door_code"),
		MessageToDriver: getStringField(addressData, "message_to_driver"),
	}

	deliveryDate := mcp.ParseString(request, "delivery_date", "")
	if deliveryDate == "" {
		return mcp.NewToolResultError("delivery_date parameter is required"), nil
	}

	timeSlot := mcp.ParseString(request, "time_slot", "")
	if timeSlot == "" {
		return mcp.NewToolResultError("time_slot parameter is required"), nil
	}

	startTime, endTime, err := willys.ValidateTimeSlot(timeSlot)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid time slot: %v", err)), nil
	}

	if err := willys.ValidateDeliveryDate(deliveryDate); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid delivery date: %v", err)), nil
	}

	availableSlots, err := h.client.GetAvailableTimeSlots(ctx, address.PostalCode)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get time slots: %v", err)), nil
	}

	if len(availableSlots) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("No delivery slots available for postal code %s", address.PostalCode)), nil
	}

	var matchedSlot *willys.TimeSlot
	for i := range availableSlots {
		slot := &availableSlots[i]
		if slot.Date == deliveryDate && slot.StartTime == startTime && slot.EndTime == endTime && slot.Available {
			matchedSlot = slot
			break
		}
	}

	if matchedSlot == nil {
		var availableTimes []string
		slotsByDate := make(map[string][]string)
		for _, slot := range availableSlots {
			if slot.Available {
				timeRange := fmt.Sprintf("%s-%s", slot.StartTime, slot.EndTime)
				slotsByDate[slot.Date] = append(slotsByDate[slot.Date], timeRange)
			}
		}

		for date, times := range slotsByDate {
			availableTimes = append(availableTimes, fmt.Sprintf("%s: %s", date, strings.Join(times, ", ")))
		}

		return mcp.NewToolResultError(fmt.Sprintf(
			"No matching time slot found for %s %s-%s. Available slots:\n%s\nPlease use get_available_time_slots tool to see all options.",
			deliveryDate, startTime, endTime, strings.Join(availableTimes, "\n"),
		)), nil
	}

	slot := *matchedSlot

	deliveryInfo, err := h.client.SetupDelivery(ctx, address, slot)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to setup delivery: %v", err)), nil
	}

	return mcp.NewToolResultJSON(deliveryInfo)
}

func (h *ToolHandler) GetAvailableTimeSlots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	postalCode := mcp.ParseString(request, "postal_code", "")
	if postalCode == "" {
		return mcp.NewToolResultError("postal_code parameter is required"), nil
	}

	slots, err := h.client.GetAvailableTimeSlots(ctx, postalCode)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get time slots: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]any{
		"slots": slots,
		"count": len(slots),
	})
}

func (h *ToolHandler) ProceedToCheckout(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	checkoutURL := h.client.GetCheckoutURL()

	return mcp.NewToolResultJSON(map[string]any{
		"checkout_url": checkoutURL,
		"message":      "Visit this URL to complete payment",
	})
}

func getStringField(m map[string]any, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
