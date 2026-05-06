package validator

import "fmt"

// ValidatePositiveInt64 проверяет, что значение положительное.
// Используется для валидации идентификаторов (productID, userID и т.п.) на границе HTTP-слоя.
func ValidatePositiveInt64(name string, v int64) error {
	if v <= 0 {
		return fmt.Errorf("%s must be a positive integer", name)
	}
	return nil
}
