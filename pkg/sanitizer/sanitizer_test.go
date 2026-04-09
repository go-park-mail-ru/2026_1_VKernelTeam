package sanitizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripHTML(t *testing.T) {
	t.Run("regular line", func(t *testing.T) {
		str := "string without scripts"
		cleanStr := StripHTML(str)
		assert.Equal(t, str, cleanStr)
	})

	t.Run("detecting dangerous line", func(t *testing.T) {
		str := `<a onblur="alert(document.сookie)" href="javascript:alert(document.сookie)">Mail.ru</a>`
		expectedStr := "Mail.ru"
		assert.Equal(t, expectedStr, StripHTML(str))
	})
}
