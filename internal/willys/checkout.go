package willys

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type (
	DeliveryAddress struct {
		FirstName       string `json:"firstName"`
		LastName        string `json:"lastName"`
		Address         string `json:"address"`
		PostalCode      string `json:"postalCode"`
		City            string `json:"city"`
		DoorCode        string `json:"doorCode,omitempty"`
		MessageToDriver string `json:"messageToDriver,omitempty"`
	}

	TimeSlot struct {
		SlotID           string  `json:"slotId"`
		Date             string  `json:"date"`
		StartTime        string  `json:"startTime"`
		EndTime          string  `json:"endTime"`
		Fee              float64 `json:"fee"`
		Available        bool    `json:"available"`
		EarliestDateTime int64   `json:"earliestDateTime"` // Unix timestamp in ms
		LatestDateTime   int64   `json:"latestDateTime"`   // Unix timestamp in ms
		RouteID          int     `json:"routeID"`
		ResourceKey      string  `json:"resourceKey"`
		ScheduleKey      string  `json:"scheduleKey"`
		PrecedingStopId  int     `json:"precedingStopId"`
		StopNumber       int     `json:"stopNumber"`
		Profitability    float64 `json:"profitability"`
	}
	DeliveryInfo struct {
		Address     DeliveryAddress `json:"address"`
		TimeSlot    TimeSlot        `json:"timeSlot"`
		PickingFee  float64         `json:"pickingFee"`
		DeliveryFee float64         `json:"deliveryFee"`
		TotalFee    float64         `json:"totalFee"`
	}
)

func (c *Client) CheckDeliverability(ctx context.Context, postalCode string) (bool, error) {
	if err := ValidatePostalCode(postalCode); err != nil {
		return false, err
	}

	path := fmt.Sprintf("%s/%s/deliverability?b2b=false", EndpointShippingDelivery, postalCode)

	resp, err := c.DoRequest(ctx, "GET", path, nil, false)
	if err != nil {
		return false, NewAPIError(0, path, "check deliverability request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var result struct {
		Deliverable bool `json:"deliverable"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, NewAPIError(resp.StatusCode, path, "failed to parse deliverability response", err)
	}

	return result.Deliverable, nil
}

func (c *Client) SetDeliveryMode(ctx context.Context) error {
	path := EndpointCartDeliveryMode + "?newSuggestedStoreId="
	resp, err := c.DoRequest(ctx, "POST", path, nil, true)
	if err != nil {
		return NewAPIError(0, path, "set delivery mode request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return NewAPIError(resp.StatusCode, path, "set delivery mode failed", nil)
	}

	return nil
}

func (c *Client) SetDeliveryAddress(ctx context.Context, address DeliveryAddress) error {
	if err := ValidateDeliveryAddress(address); err != nil {
		return err
	}

	params := url.Values{}
	params.Set("firstName", address.FirstName)
	params.Set("lastName", address.LastName)
	params.Set("addressLine1", address.Address) // API uses addressLine1, not address
	params.Set("addressLine2", "")
	params.Set("postalCode", address.PostalCode)
	params.Set("town", address.City) // API uses town, not city
	params.Set("cellphone", "")
	params.Set("longitude", "")
	params.Set("latitude", "")

	if address.DoorCode != "" {
		params.Set("doorCode", address.DoorCode)
	}
	if address.MessageToDriver != "" {
		params.Set("messageToDriver", address.MessageToDriver)
	}

	path := fmt.Sprintf("%s?%s", EndpointCartDeliveryAddress, params.Encode())
	resp, err := c.DoRequest(ctx, "POST", path, nil, true)
	if err != nil {
		return NewAPIError(0, path, "set delivery address request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return NewAPIError(resp.StatusCode, path, "set delivery address failed", nil)
	}

	postalPath := fmt.Sprintf("%s?postalCode=%s", EndpointCartPostalCode, address.PostalCode)
	postalResp, err := c.DoRequest(ctx, "POST", postalPath, nil, true)
	if err != nil {
		return NewAPIError(0, postalPath, "set postal code request failed", err)
	}
	defer postalResp.Body.Close()

	if postalResp.StatusCode != http.StatusOK && postalResp.StatusCode != http.StatusNoContent {
		return NewAPIError(postalResp.StatusCode, postalPath, "set postal code failed", nil)
	}

	return nil
}

func (c *Client) GetAvailableTimeSlots(ctx context.Context, postalCode string) ([]TimeSlot, error) {
	if err := ValidatePostalCode(postalCode); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s?postalCode=%s&b2b=false", EndpointSlotHomeDelivery, postalCode)

	resp, err := c.DoRequest(ctx, "GET", path, nil, false)
	if err != nil {
		return nil, NewAPIError(0, path, "get time slots request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewAPIError(resp.StatusCode, path, "get time slots failed", nil)
	}

	var result struct {
		Isocode string `json:"isocode"`
		Slots   []struct {
			Code          string `json:"code"`
			StartTime     int64  `json:"startTime"` // Unix timestamp in milliseconds
			EndTime       int64  `json:"endTime"`   // Unix timestamp in milliseconds
			FormattedTime string `json:"formattedTime"`
			DeliveryCost  struct {
				Value float64 `json:"value"`
			} `json:"deliveryCost"`
			Available                  bool `json:"available"`
			TmsDeliveryWindowReference struct {
				EarliestDateTime int64   `json:"earliestDateTime"`
				LatestDateTime   int64   `json:"latestDateTime"`
				RouteID          int     `json:"routeID"`
				ResourceKey      string  `json:"resourceKey"`
				ScheduleKey      string  `json:"scheduleKey"`
				PrecedingStopId  int     `json:"precedingStopId"`
				StopNumber       int     `json:"stopNumber"`
				Profitability    float64 `json:"profitability"`
			} `json:"tmsDeliveryWindowReference"`
		} `json:"slots"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, NewAPIError(resp.StatusCode, path, "failed to parse time slots response", err)
	}

	slots := make([]TimeSlot, 0)
	for _, s := range result.Slots {
		startTimeObj := time.Unix(s.StartTime/1000, 0)
		endTimeObj := time.Unix(s.EndTime/1000, 0)

		slot := TimeSlot{
			SlotID:           s.Code,
			Date:             startTimeObj.Format("2006-01-02"),
			StartTime:        startTimeObj.Format("15:04"),
			EndTime:          endTimeObj.Format("15:04"),
			Fee:              s.DeliveryCost.Value,
			Available:        s.Available,
			EarliestDateTime: s.TmsDeliveryWindowReference.EarliestDateTime,
			LatestDateTime:   s.TmsDeliveryWindowReference.LatestDateTime,
			RouteID:          s.TmsDeliveryWindowReference.RouteID,
			ResourceKey:      s.TmsDeliveryWindowReference.ResourceKey,
			ScheduleKey:      s.TmsDeliveryWindowReference.ScheduleKey,
			PrecedingStopId:  s.TmsDeliveryWindowReference.PrecedingStopId,
			StopNumber:       s.TmsDeliveryWindowReference.StopNumber,
			Profitability:    s.TmsDeliveryWindowReference.Profitability,
		}
		slots = append(slots, slot)
	}

	return slots, nil
}

func (c *Client) SelectTimeSlot(ctx context.Context, slot TimeSlot) error {
	reqData := struct {
		EarliestDateTime int64   `json:"earliestDateTime"`
		LatestDateTime   int64   `json:"latestDateTime"`
		RouteID          int     `json:"routeID"`
		ResourceKey      string  `json:"resourceKey"`
		ScheduleKey      string  `json:"scheduleKey"`
		PrecedingStopId  int     `json:"precedingStopId"`
		StopNumber       int     `json:"stopNumber"`
		Profitability    float64 `json:"profitability"`
	}{
		EarliestDateTime: slot.EarliestDateTime,
		LatestDateTime:   slot.LatestDateTime,
		RouteID:          slot.RouteID,
		ResourceKey:      slot.ResourceKey,
		ScheduleKey:      slot.ScheduleKey,
		PrecedingStopId:  slot.PrecedingStopId,
		StopNumber:       slot.StopNumber,
		Profitability:    slot.Profitability,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return NewAPIError(0, EndpointSlotInCart, "failed to marshal time slot request", err)
	}

	path := fmt.Sprintf("%s/%s?isTmsSlot=true", EndpointSlotInCart, url.QueryEscape(slot.SlotID))
	resp, err := c.DoRequest(ctx, "POST", path, bytes.NewReader(jsonData), true)
	if err != nil {
		return NewAPIError(0, path, "select time slot request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return NewAPIError(resp.StatusCode, path, "select time slot failed", nil)
	}

	return nil
}

func (c *Client) GetCheckoutURL() string {
	return c.baseURL + EndpointCheckout
}

func (c *Client) SetupDelivery(ctx context.Context, address DeliveryAddress, slot TimeSlot) (*DeliveryInfo, error) {
	available, err := c.CheckDeliverability(ctx, address.PostalCode)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, NewValidationError("postal_code", fmt.Sprintf("delivery not available for postal code %s", address.PostalCode))
	}

	if err := c.SetDeliveryMode(ctx); err != nil {
		return nil, err
	}

	if err := c.SetDeliveryAddress(ctx, address); err != nil {
		return nil, err
	}

	if err := c.SelectTimeSlot(ctx, slot); err != nil {
		return nil, err
	}

	deliveryInfo := &DeliveryInfo{
		Address:     address,
		TimeSlot:    slot,
		PickingFee:  DefaultPickingFee,
		DeliveryFee: slot.Fee,
		TotalFee:    DefaultPickingFee + slot.Fee,
	}

	return deliveryInfo, nil
}
