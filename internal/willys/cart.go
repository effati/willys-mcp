package willys

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

type (
	CartItem struct {
		ProductCode string  `json:"code"`
		Name        string  `json:"name"`
		Quantity    int     `json:"quantity"`
		Price       float64 `json:"price"`
		TotalPrice  float64 `json:"totalPrice"`
		ImageURL    string  `json:"imageUrl"`
	}

	CartSummary struct {
		Items       []CartItem `json:"items"`
		TotalPrice  float64    `json:"totalPrice"`
		ItemCount   int        `json:"itemCount"`
		DeliveryFee float64    `json:"deliveryFee"`
		PickingFee  float64    `json:"pickingFee"`
		FinalTotal  float64    `json:"finalTotal"`
	}

	AddToCartRequest struct {
		Products []AddToCartRequestProduct `json:"products"`
	}

	AddToCartRequestProduct struct {
		ProductCodePost     string `json:"productCodePost"`
		Qty                 int    `json:"qty"`
		PickUnit            string `json:"pickUnit"`
		HideDiscountToolTip bool   `json:"hideDiscountToolTip"`
		NoReplacementFlag   bool   `json:"noReplacementFlag"`
	}

	// Prices can be a string, number, or an object with a "value" field
	FlexiblePrice struct {
		value any
	}

	CartProductData struct {
		Code     string        `json:"code"`
		Name     string        `json:"name"`
		Quantity int           `json:"quantity"`
		Price    FlexiblePrice `json:"price"` // Can be string, number, or {value: number}
		Image    struct {
			URL string `json:"url"`
		} `json:"image"`
	}

	CartResponseData struct {
		Products    []CartProductData `json:"products"`
		TotalPrice  FlexiblePrice     `json:"totalPrice"`  // Can be string or number
		DeliveryFee FlexiblePrice     `json:"deliveryFee"` // Can be string or number
		PickingFee  FlexiblePrice     `json:"pickingFee"`  // Can be string or number
	}
)

func (c *Client) AddToCart(ctx context.Context, productCode string, quantity int) (*CartSummary, error) {
	if err := ValidateProductCode(productCode); err != nil {
		return nil, err
	}
	if err := ValidateQuantity(quantity); err != nil {
		return nil, err
	}

	req := AddToCartRequest{
		Products: []AddToCartRequestProduct{
			{
				productCode,
				quantity,
				"pieces",
				false,
				false,
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, NewAPIError(0, EndpointCartAddProducts, "failed to marshal add to cart request", err)
	}

	resp, err := c.DoRequest(ctx, "POST", EndpointCartAddProducts, bytes.NewReader(jsonData), true)
	if err != nil {
		return nil, NewAPIError(0, EndpointCartAddProducts, "add to cart request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, NewNotFoundError("product", productCode)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, NewAPIError(resp.StatusCode, EndpointCartAddProducts, "add to cart failed", nil)
	}

	return c.GetCart(ctx)
}

func (fp *FlexiblePrice) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	fp.value = v
	return nil
}

func (fp FlexiblePrice) Value() any {
	return fp.value
}

func parsePrice(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		if val == "" {
			return 0
		}
		price, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0
		}
		return price
	case map[string]any:
		if valueField, ok := val["value"]; ok {
			return parsePrice(valueField)
		}
		return 0
	default:
		return 0
	}
}

func (c *Client) GetCart(ctx context.Context) (*CartSummary, error) {
	resp, err := c.DoRequest(ctx, "GET", EndpointCart, nil, false)
	if err != nil {
		return nil, NewAPIError(0, EndpointCart, "get cart request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewAPIError(resp.StatusCode, EndpointCart, "get cart failed", nil)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewAPIError(resp.StatusCode, EndpointCart, "failed to read cart response", err)
	}

	var cartData CartResponseData

	if err := json.Unmarshal(body, &cartData); err != nil {
		return nil, NewAPIError(resp.StatusCode, EndpointCart, "failed to parse cart response", err)
	}

	totalPrice := parsePrice(cartData.TotalPrice.Value())
	deliveryFee := parsePrice(cartData.DeliveryFee.Value())
	pickingFee := parsePrice(cartData.PickingFee.Value())

	items := make([]CartItem, 0, len(cartData.Products))
	itemCount := 0

	for _, product := range cartData.Products {
		itemPrice := parsePrice(product.Price.Value())
		cartItem := CartItem{
			product.Code,
			product.Name,
			product.Quantity,
			itemPrice,
			itemPrice * float64(product.Quantity),
			product.Image.URL,
		}
		items = append(items, cartItem)
		itemCount += product.Quantity
	}

	finalTotal := totalPrice + deliveryFee + pickingFee

	return &CartSummary{
		items,
		totalPrice,
		itemCount,
		deliveryFee,
		pickingFee,
		finalTotal,
	}, nil
}

func (c *Client) RemoveFromCart(ctx context.Context, productCode string, quantity int) (*CartSummary, error) {
	var newQty int

	if quantity <= 0 {
		newQty = 0
	} else {
		currentCart, err := c.GetCart(ctx)
		if err != nil {
			return nil, err
		}

		currentQty := 0
		found := false
		for _, item := range currentCart.Items {
			if item.ProductCode == productCode {
				currentQty = item.Quantity
				found = true
				break
			}
		}

		if !found {
			return currentCart, nil
		}

		newQty = max(currentQty-quantity, 0)
	}

	req := AddToCartRequest{
		Products: []AddToCartRequestProduct{
			{
				productCode,
				newQty,
				"pieces",
				false,
				false,
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, NewAPIError(0, EndpointCartAddProducts, "failed to marshal remove from cart request", err)
	}

	resp, err := c.DoRequest(ctx, "POST", EndpointCartAddProducts, bytes.NewReader(jsonData), true)
	if err != nil {
		return nil, NewAPIError(0, EndpointCartAddProducts, "remove from cart request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, NewAPIError(resp.StatusCode, EndpointCartAddProducts, "remove from cart failed", nil)
	}

	return c.GetCart(ctx)
}

func (c *Client) ClearCart(ctx context.Context) error {
	resp, err := c.DoRequest(ctx, "DELETE", EndpointCart, nil, true)
	if err != nil {
		return NewAPIError(0, EndpointCart, "clear cart request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return NewAPIError(resp.StatusCode, EndpointCart, "clear cart failed", nil)
	}

	return nil
}
