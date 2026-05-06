package translit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateVariants_LatinInput(t *testing.T) {
	original, translit, layout := GenerateVariants("iPhone")

	assert.Equal(t, "iphone", original)
	// "ph" даёт "ф" по longest-match
	assert.Equal(t, "ифоне", translit)
	assert.Equal(t, "шзрщту", layout)
}

func TestGenerateVariants_CyrillicInput(t *testing.T) {
	original, translit, layout := GenerateVariants("Айфон")

	assert.Equal(t, "айфон", original)
	assert.Equal(t, "ayfon", translit)
	assert.Equal(t, "fqajy", layout)
}

func TestGenerateVariants_Trim(t *testing.T) {
	original, _, _ := GenerateVariants("   apple   ")
	assert.Equal(t, "apple", original)
}

func TestGenerateVariants_NonLetters(t *testing.T) {
	original, translit, layout := GenerateVariants("123 !!!")
	assert.Equal(t, "123 !!!", original)
	assert.Equal(t, original, translit)
	assert.Equal(t, original, layout)
}

func TestGenerateVariants_LongestMatchWins(t *testing.T) {
	_, translit, _ := GenerateVariants("shchuka")
	assert.Equal(t, "щука", translit)
}

func TestGenerateVariants_CyrillicMultichar(t *testing.T) {
	_, translit, _ := GenerateVariants("щука")
	assert.Equal(t, "shchuka", translit)
}

func TestGenerateVariants_SoftHardSignsDropped(t *testing.T) {
	_, translit, _ := GenerateVariants("объять")
	assert.Equal(t, "obyat", translit)
}

func TestGenerateVariants_LatinWithDigits(t *testing.T) {
	original, translit, layout := GenerateVariants("apple5")
	assert.Equal(t, "apple5", original)
	assert.Equal(t, "аппле5", translit)
	assert.Equal(t, "фззду5", layout)
}

func TestGenerateVariants_CyrillicWithDigits(t *testing.T) {
	_, _, layout := GenerateVariants("айфон5")
	assert.Equal(t, "fqajy5", layout)
}

func TestGenerateVariants_EmptyInput(t *testing.T) {
	original, translit, layout := GenerateVariants("")
	assert.Empty(t, original)
	assert.Empty(t, translit)
	assert.Empty(t, layout)
}
