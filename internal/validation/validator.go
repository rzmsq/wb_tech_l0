// Package validation обеспечивает валидацию структур и идентификаторов заказов.
package validation

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

var v = validator.New()

// ValidateOrder проверяет, соответствует ли структура заказа правилам валидации.
func ValidateOrder(o interface{}) error {
	if err := v.Struct(o); err != nil {
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			return err
		}
		// Aggregate readable message
		out := "validation failed:"
		for _, fe := range err.(validator.ValidationErrors) {
			out += fmt.Sprintf(" %s(%s %s)", fe.Field(), fe.Tag(), fe.Param())
		}
		return fmt.Errorf(out)
	}
	return nil
}

// ValidateOrderID проверяет, соответствует ли идентификатор заказа допустимым символам (буквы и цифры).
func ValidateOrderID(id string) bool {
	if len(id) == 0 {
		return false
	}

	for _, r := range id {
		if !(r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r == '-') {
			return false
		}
	}
	return true
}
