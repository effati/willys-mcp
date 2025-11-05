package willys

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	postalCodeRegex = regexp.MustCompile(`^\d{5}$|^\d{3}\s?\d{2}$`)

	productCodeRegex = regexp.MustCompile(`^\d+_(ST|KG)$`)

	timeFormatRegex = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)
)

const (
	maxAddressLength     = 100
	maxNameLength        = 50
	maxCityLength        = 50
	maxDoorCodeLength    = 20
	maxMessageLength     = 500
	maxDeliveryDaysAhead = 14 // Maximum days ahead for delivery scheduling
)

func ValidatePostalCode(postalCode string) error {
	if postalCode == "" {
		return NewValidationError("postal_code", "cannot be empty")
	}
	if !postalCodeRegex.MatchString(postalCode) {
		return NewValidationError("postal_code", "invalid format (expected: 12345 or 123 45)")
	}
	return nil
}

func ValidateProductCode(code string) error {
	if code == "" {
		return NewValidationError("product_code", "cannot be empty")
	}
	if !productCodeRegex.MatchString(code) {
		return NewValidationError("product_code", "invalid format (expected: 123456_ST or 123456_KG)")
	}
	return nil
}

func ValidateQuantity(quantity int) error {
	if quantity < 1 {
		return NewValidationError("quantity", "must be at least 1")
	}
	if quantity > 999 {
		return NewValidationError("quantity", "max 999")
	}
	return nil
}

func ValidateDeliveryAddress(address DeliveryAddress) error {
	if address.FirstName == "" {
		return NewValidationError("first_name", "required")
	}
	if len(address.FirstName) > maxNameLength {
		return NewValidationError("first_name", fmt.Sprintf("max %d characters", maxNameLength))
	}
	if address.LastName == "" {
		return NewValidationError("last_name", "required")
	}
	if len(address.LastName) > maxNameLength {
		return NewValidationError("last_name", fmt.Sprintf("max %d characters", maxNameLength))
	}
	if address.Address == "" {
		return NewValidationError("address", "required")
	}
	if len(address.Address) > maxAddressLength {
		return NewValidationError("address", fmt.Sprintf("max %d characters", maxAddressLength))
	}
	if address.City == "" {
		return NewValidationError("city", "required")
	}
	if len(address.City) > maxCityLength {
		return NewValidationError("city", fmt.Sprintf("max %d characters", maxCityLength))
	}
	if len(address.DoorCode) > maxDoorCodeLength {
		return NewValidationError("door_code", fmt.Sprintf("max %d characters", maxDoorCodeLength))
	}
	if len(address.MessageToDriver) > maxMessageLength {
		return NewValidationError("message_to_driver", fmt.Sprintf("max %d characters", maxMessageLength))
	}
	if err := ValidatePostalCode(address.PostalCode); err != nil {
		return err
	}
	return nil
}

func ValidateDeliveryDate(dateStr string) error {
	if dateStr == "" {
		return NewValidationError("delivery_date", "cannot be empty")
	}

	deliveryDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return NewValidationError("delivery_date", "invalid format (expected: YYYY-MM-DD)")
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if deliveryDate.Before(today) {
		return NewValidationError("delivery_date", "cannot be in the past")
	}

	maxDate := today.AddDate(0, 0, maxDeliveryDaysAhead)
	if deliveryDate.After(maxDate) {
		return NewValidationError("delivery_date", fmt.Sprintf("max %d days ahead", maxDeliveryDaysAhead))
	}

	return nil
}

func ValidateTimeSlot(timeSlot string) (string, string, error) {
	if timeSlot == "" {
		return "", "", NewValidationError("time_slot", "cannot be empty")
	}

	parts := strings.Split(timeSlot, "-")
	if len(parts) != 2 {
		return "", "", NewValidationError("time_slot", "invalid format (expected: HH:MM-HH:MM)")
	}

	startTime := strings.TrimSpace(parts[0])
	endTime := strings.TrimSpace(parts[1])

	if !timeFormatRegex.MatchString(startTime) {
		return "", "", NewValidationError("time_slot", fmt.Sprintf("invalid start time: %s", startTime))
	}
	if !timeFormatRegex.MatchString(endTime) {
		return "", "", NewValidationError("time_slot", fmt.Sprintf("invalid end time: %s", endTime))
	}

	start, _ := time.Parse("15:04", startTime)
	end, _ := time.Parse("15:04", endTime)

	if !end.After(start) {
		return "", "", NewValidationError("time_slot", "end time must be after start time")
	}

	return startTime, endTime, nil
}
