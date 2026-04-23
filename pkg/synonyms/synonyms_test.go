package synonyms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpand_KnownBrand(t *testing.T) {
	result := Expand("iphone")
	assert.Contains(t, result, "айфон")
	assert.Contains(t, result, "aifon")
	assert.NotContains(t, result, "iphone")
}

func TestExpand_CyrillicInput(t *testing.T) {
	result := Expand("найк")
	assert.Contains(t, result, "nike")
	assert.Contains(t, result, "нике")
	assert.NotContains(t, result, "найк")
}

func TestExpand_Unknown(t *testing.T) {
	result := Expand("unknownword")
	assert.Nil(t, result)
}

func TestExpand_CaseInsensitive(t *testing.T) {
	result := Expand("IPhone")
	assert.Contains(t, result, "айфон")
}

func TestExpandAll_SingleWord(t *testing.T) {
	result := ExpandAll("xiaomi")
	assert.Contains(t, result, "сяоми")
	assert.Contains(t, result, "ксиоми")
}

func TestExpandAll_MultiWord(t *testing.T) {
	result := ExpandAll("nike кроссовки")
	assert.Contains(t, result, "найк")
	assert.Contains(t, result, "нике")
}

func TestExpandAll_FullPhrase(t *testing.T) {
	result := ExpandAll("louis vuitton")
	assert.Contains(t, result, "луи виттон")
}

func TestExpandAll_NoMatch(t *testing.T) {
	result := ExpandAll("обычный запрос без брендов")
	assert.Empty(t, result)
}
