package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePositiveInt64(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		input     int64
		wantErr   bool
		wantMsg   string
	}{
		{"positive", "product_id", 1, false, ""},
		{"large positive", "user_id", 1<<62 - 1, false, ""},
		{"zero", "product_id", 0, true, "product_id must be a positive integer"},
		{"negative", "user_id", -1, true, "user_id must be a positive integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePositiveInt64(tt.fieldName, tt.input)
			if !tt.wantErr {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.wantMsg)
		})
	}
}
