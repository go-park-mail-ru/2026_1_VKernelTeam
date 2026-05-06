package synonyms

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpand(t *testing.T) {
	t.Run("known word returns siblings without itself", func(t *testing.T) {
		got := Expand("iphone")
		assert.NotContains(t, got, "iphone")
		assert.ElementsMatch(t, []string{"айфон", "aifon", "aiphone"}, got)
	})

	t.Run("case and whitespace are normalized", func(t *testing.T) {
		got := Expand("  IPhone  ")
		assert.ElementsMatch(t, []string{"айфон", "aifon", "aiphone"}, got)
	})

	t.Run("cyrillic input finds latin sibling", func(t *testing.T) {
		got := Expand("айфон")
		assert.Contains(t, got, "iphone")
	})

	t.Run("unknown returns nil", func(t *testing.T) {
		assert.Nil(t, Expand("unknownbrand"))
	})

	t.Run("empty returns nil", func(t *testing.T) {
		assert.Nil(t, Expand(""))
	})
}

func TestExpandAll(t *testing.T) {
	t.Run("single known word", func(t *testing.T) {
		got := ExpandAll("samsung")
		assert.ElementsMatch(t, []string{"самсунг", "самсун"}, got)
	})

	t.Run("multi-word phrase expands phrase first", func(t *testing.T) {
		got := ExpandAll("louis vuitton")
		assert.Contains(t, got, "луи виттон")
		assert.Contains(t, got, "луи витон")
	})

	t.Run("multi-word with one known word", func(t *testing.T) {
		got := ExpandAll("apple watch")
		assert.Contains(t, got, "эпл")
		assert.Contains(t, got, "апле")
	})

	t.Run("results are deduplicated", func(t *testing.T) {
		got := ExpandAll("apple apple")
		seen := make(map[string]int)
		for _, s := range got {
			seen[s]++
		}
		for s, n := range seen {
			assert.Equal(t, 1, n, "duplicate %q in result", s)
		}
	})

	t.Run("unknown returns empty", func(t *testing.T) {
		assert.Empty(t, ExpandAll("totally unknown query"))
	})

	t.Run("results are stable order across calls", func(t *testing.T) {
		first := ExpandAll("nike")
		second := ExpandAll("nike")
		sort.Strings(first)
		sort.Strings(second)
		assert.Equal(t, first, second)
	})
}
