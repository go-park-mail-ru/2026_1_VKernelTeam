package translit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateVariants_Cyrillic(t *testing.T) {
	original, translitVar, layoutVar := GenerateVariants("макбук")
	assert.Equal(t, "макбук", original)
	assert.Equal(t, "makbuk", translitVar)
	assert.Equal(t, "vfr,er", layoutVar)
}

func TestGenerateVariants_Latin_Macbook(t *testing.T) {
	original, translitVar, layoutVar := GenerateVariants("macbook")
	assert.Equal(t, "macbook", original)
	assert.Equal(t, "макбук", translitVar)
	assert.Equal(t, "ьфсищщл", layoutVar)
}

func TestGenerateVariants_Latin_Samsung(t *testing.T) {
	_, translitVar, _ := GenerateVariants("samsung")
	assert.Equal(t, "самсунг", translitVar)
}

func TestGenerateVariants_Latin_Bosch(t *testing.T) {
	_, translitVar, _ := GenerateVariants("bosch")
	assert.Equal(t, "бош", translitVar)
}

func TestGenerateVariants_Latin_Reebok(t *testing.T) {
	_, translitVar, _ := GenerateVariants("reebok")
	assert.Equal(t, "рибок", translitVar)
}

func TestGenerateVariants_Latin_Nike(t *testing.T) {
	_, translitVar, _ := GenerateVariants("nike")
	assert.Equal(t, "нике", translitVar)
}

func TestGenerateVariants_Latin_Phone(t *testing.T) {
	_, translitVar, _ := GenerateVariants("phone")
	assert.Equal(t, "фоне", translitVar)
}

func TestGenerateVariants_LayoutSwitch_QwertyToCyrillic(t *testing.T) {
	_, _, layoutVar := GenerateVariants("vfr,er")
	assert.Equal(t, "макбук", layoutVar)
}

func TestGenerateVariants_LayoutSwitch_CyrToQwerty(t *testing.T) {
	_, _, layoutVar := GenerateVariants("про")
	assert.Equal(t, "ghj", layoutVar)
}

func TestGenerateVariants_CaseInsensitive(t *testing.T) {
	original, _, _ := GenerateVariants("MacBook")
	assert.Equal(t, "macbook", original)
}

func TestGenerateVariants_Whitespace(t *testing.T) {
	original, _, _ := GenerateVariants("  макбук  ")
	assert.Equal(t, "макбук", original)
}

func TestGenerateVariants_CyrToLat_iPhone(t *testing.T) {
	_, translitVar, _ := GenerateVariants("айфон")
	assert.Equal(t, "ayfon", translitVar)
}

func TestGenerateVariants_CyrToLat_MultiChar(t *testing.T) {
	_, translitVar, _ := GenerateVariants("щётка")
	assert.Contains(t, translitVar, "shch")
}

func TestGenerateVariants_Roundtrip_Adidas(t *testing.T) {
	_, translitVar, _ := GenerateVariants("адидас")
	assert.Equal(t, "adidas", translitVar)
}

func TestGenerateVariants_LayoutRoundtrip(t *testing.T) {
	_, _, layoutVar := GenerateVariants("ghj")
	assert.Equal(t, "про", layoutVar)
}
