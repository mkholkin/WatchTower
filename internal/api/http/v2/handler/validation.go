package handler

import (
	"fmt"
	"unicode/utf8"

	"github.com/google/uuid"
)

func validateRequiredText(value, field string, maxLength int) string {
	length := utf8.RuneCountInString(value)
	if length < 1 || length > maxLength {
		return fmt.Sprintf("%s length must be between 1 and %d characters", field, maxLength)
	}
	return ""
}

func validateOptionalText(value *string, field string, maxLength int) string {
	if value != nil && utf8.RuneCountInString(*value) > maxLength {
		return fmt.Sprintf("%s length must not exceed %d characters", field, maxLength)
	}
	return ""
}

func hasDuplicateUUIDs(values []uuid.UUID) bool {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
