package sanitizer

import "github.com/microcosm-cc/bluemonday"

// policy запрещает весь HTML — только plain text.
// StrictPolicy удаляет все теги и атрибуты через HTML-парсер,
// поэтому обходы вроде <<script> или <scr\0ipt> тоже не проходят
var policy = bluemonday.StrictPolicy()

// StripHTML удаляет все HTML-теги из строки, оставляя только текст
// "<b>Hello</b>" -> "Hello"
// "<script>alert(1)</script>" -> "alert(1)"
// "<img src=x onerror=alert(1)>" -> ""
func StripHTML(s string) string {
	return policy.Sanitize(s)
}
