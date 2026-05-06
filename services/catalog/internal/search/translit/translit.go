package translit

import (
	"strings"
	"unicode"
)

// cyrToLat — кириллица → латиница (фонетическая).
// Многосимвольные маппинги идут первыми для корректной обработки.
var cyrToLat = []struct {
	cyr string
	lat string
}{
	{"щ", "shch"},
	{"ш", "sh"},
	{"ч", "ch"},
	{"ж", "zh"},
	{"х", "kh"},
	{"ц", "ts"},
	{"ю", "yu"},
	{"я", "ya"},
	{"а", "a"},
	{"б", "b"},
	{"в", "v"},
	{"г", "g"},
	{"д", "d"},
	{"е", "e"},
	{"з", "z"},
	{"и", "i"},
	{"й", "y"},
	{"к", "k"},
	{"л", "l"},
	{"м", "m"},
	{"н", "n"},
	{"о", "o"},
	{"п", "p"},
	{"р", "r"},
	{"с", "s"},
	{"т", "t"},
	{"у", "u"},
	{"ф", "f"},
	{"ы", "y"},
	{"э", "e"},
	{"ё", "yo"},
	{"ь", ""},
	{"ъ", ""},
}

// latToCyr — латиница → кириллица (фонетическая).
// Отдельная таблица, т.к. обратное преобразование не симметрично.
// Многосимвольные маппинги идут первыми (longest match wins).
var latToCyr = []struct {
	lat string
	cyr string
}{
	{"shch", "щ"},
	{"sch", "ш"},
	{"sh", "ш"},
	{"ch", "ч"},
	{"zh", "ж"},
	{"ph", "ф"},
	{"th", "т"},
	{"kh", "х"},
	{"ts", "ц"},
	{"yu", "ю"},
	{"ya", "я"},
	{"yo", "ё"},
	{"oo", "у"},
	{"ee", "и"},
	{"a", "а"},
	{"b", "б"},
	{"c", "к"},
	{"d", "д"},
	{"e", "е"},
	{"f", "ф"},
	{"g", "г"},
	{"h", "х"},
	{"i", "и"},
	{"j", "дж"},
	{"k", "к"},
	{"l", "л"},
	{"m", "м"},
	{"n", "н"},
	{"o", "о"},
	{"p", "п"},
	{"q", "к"},
	{"r", "р"},
	{"s", "с"},
	{"t", "т"},
	{"u", "у"},
	{"v", "в"},
	{"w", "в"},
	{"x", "кс"},
	{"y", "и"},
	{"z", "з"},
}

// QWERTY → ЙЦУКЕН keyboard layout mapping
var qwertyToCyr = map[rune]rune{
	'q': 'й', 'w': 'ц', 'e': 'у', 'r': 'к', 't': 'е', 'y': 'н', 'u': 'г', 'i': 'ш', 'o': 'щ', 'p': 'з',
	'[': 'х', ']': 'ъ',
	'a': 'ф', 's': 'ы', 'd': 'в', 'f': 'а', 'g': 'п', 'h': 'р', 'j': 'о', 'k': 'л', 'l': 'д',
	';': 'ж', '\'': 'э',
	'z': 'я', 'x': 'ч', 'c': 'с', 'v': 'м', 'b': 'и', 'n': 'т', 'm': 'ь',
	',': 'б', '.': 'ю',
}

var cyrToQwerty map[rune]rune

func init() {
	cyrToQwerty = make(map[rune]rune, len(qwertyToCyr))
	for lat, cyr := range qwertyToCyr {
		cyrToQwerty[cyr] = lat
	}
}

// GenerateVariants возвращает три варианта запроса для поиска:
// original, phonetic transliteration, keyboard layout switch.
func GenerateVariants(query string) (original, translit, layout string) {
	query = strings.TrimSpace(query)
	lower := strings.ToLower(query)
	original = lower

	if isCyrillic(lower) {
		translit = cyrillicToLatin(lower)
		layout = cyrillicToQwerty(lower)
	} else if isLatin(lower) {
		translit = latinToCyrillic(lower)
		layout = qwertyToCyrillic(lower)
	} else {
		translit = lower
		layout = lower
	}

	return original, translit, layout
}

func isCyrillic(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	return false
}

func isLatin(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Latin, r) {
			return true
		}
	}
	return false
}

func cyrillicToLatin(s string) string {
	result := s
	for _, pair := range cyrToLat {
		result = strings.ReplaceAll(result, pair.cyr, pair.lat)
	}
	return result
}

// latinToCyrillic выполняет посимвольную замену с приоритетом длинных последовательностей.
// Используется отдельная таблица latToCyr вместо реверса cyrToLat.
func latinToCyrillic(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)

	i := 0
	runes := []rune(s)
	for i < len(runes) {
		matched := false
		for _, pair := range latToCyr {
			pLen := len([]rune(pair.lat))
			if i+pLen <= len(runes) {
				candidate := string(runes[i : i+pLen])
				if candidate == pair.lat {
					b.WriteString(pair.cyr)
					i += pLen
					matched = true
					break
				}
			}
		}
		if !matched {
			b.WriteRune(runes[i])
			i++
		}
	}
	return b.String()
}

func qwertyToCyrillic(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if mapped, ok := qwertyToCyr[r]; ok {
			b.WriteRune(mapped)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func cyrillicToQwerty(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if mapped, ok := cyrToQwerty[r]; ok {
			b.WriteRune(mapped)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
