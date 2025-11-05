package willys

import (
	"context"
	"io"
	"net/http"
)

const (
	EndpointLogin               = "/login"
	EndpointCSRFToken           = "/axfood/rest/csrf-token"
	EndpointCustomer            = "/axfood/rest/customer"
	EndpointCart                = "/axfood/rest/cart"
	EndpointCartAddProducts     = "/axfood/rest/cart/addProducts"
	EndpointCartDeliveryMode    = "/axfood/rest/cart/delivery-mode/homeDelivery"
	EndpointCartDeliveryAddress = "/axfood/rest/cart/delivery-address"
	EndpointCartPostalCode      = "/axfood/rest/cart/postal-code"
	EndpointSearch              = "/search"
	EndpointSlotHomeDelivery    = "/axfood/rest/slot/homeDelivery"
	EndpointSlotInCart          = "/axfood/rest/slot/slotInCart"
	EndpointShippingDelivery    = "/axfood/rest/shipping/delivery"
	EndpointCheckout            = "/kassa"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type WillysAPI interface {
	Login(ctx context.Context, username, password string) error
	GetCustomerInfo(ctx context.Context) (*CustomerInfo, error)
	IsAuthenticated() bool

	SearchProducts(ctx context.Context, query string, page, size int, prefs *SearchPreferences) ([]Product, error)

	AddToCart(ctx context.Context, productCode string, quantity int) (*CartSummary, error)
	GetCart(ctx context.Context) (*CartSummary, error)
	RemoveFromCart(ctx context.Context, productCode string, quantity int) (*CartSummary, error)
	ClearCart(ctx context.Context) error

	CheckDeliverability(ctx context.Context, postalCode string) (bool, error)
	SetDeliveryMode(ctx context.Context) error
	SetDeliveryAddress(ctx context.Context, address DeliveryAddress) error
	GetAvailableTimeSlots(ctx context.Context, postalCode string) ([]TimeSlot, error)
	SelectTimeSlot(ctx context.Context, slot TimeSlot) error
	SetupDelivery(ctx context.Context, address DeliveryAddress, slot TimeSlot) (*DeliveryInfo, error)
	GetCheckoutURL() string

	GetCSRFToken() (string, error)
	FetchCSRFToken() (string, error)
	DoRequest(ctx context.Context, method, path string, body io.Reader, needsCSRF bool) (*http.Response, error)
}

var _ WillysAPI = (*Client)(nil)
