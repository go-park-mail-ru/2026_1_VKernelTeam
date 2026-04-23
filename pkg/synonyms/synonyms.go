package synonyms

import "strings"

// synonymGroups содержит группы синонимов для брендов и сложных заимствований.
// Каждая группа — это набор написаний одного и того же слова.
var synonymGroups = [][]string{
	{"iphone", "айфон", "aifon", "aiphone"},
	{"apple", "апле", "эпл", "эппл"},
	{"xiaomi", "сяоми", "ксиоми", "ксиаоми", "шаоми"},
	{"samsung", "самсунг", "самсун"},
	{"huawei", "хуавей", "хуавэй"},
	{"honor", "хонор"},
	{"realme", "реалми", "риалми"},
	{"nokia", "нокиа", "нокия"},
	{"sony", "сони"},
	{"lenovo", "леново"},
	{"asus", "асус"},
	{"acer", "асер", "эйсер"},
	{"dell", "делл"},
	{"hp", "эйчпи"},
	{"macbook", "макбук"},
	{"playstation", "плейстейшн", "плейстейшен"},
	{"xbox", "иксбокс"},
	{"nintendo", "нинтендо"},
	{"nike", "найк", "найки", "нике"},
	{"adidas", "адидас"},
	{"reebok", "рибок"},
	{"puma", "пума"},
	{"gucci", "гуччи", "гучи"},
	{"versace", "версаче"},
	{"chanel", "шанель"},
	{"dior", "диор"},
	{"prada", "прада"},
	{"louis vuitton", "луи виттон", "луи витон"},
	{"balenciaga", "баленсиага", "баленциага"},
	{"zara", "зара"},
	{"ikea", "икея", "икеа"},
	{"bosch", "бош"},
	{"siemens", "сименс"},
	{"dyson", "дайсон"},
	{"philips", "филипс"},
	{"panasonic", "панасоник"},
	{"toyota", "тойота"},
	{"hyundai", "хендай", "хундай", "хюндай"},
	{"mercedes", "мерседес"},
	{"bmw", "бмв"},
	{"volkswagen", "фольксваген"},
	{"peugeot", "пежо"},
	{"renault", "рено"},
	{"chevrolet", "шевроле"},
	{"porsche", "порше"},
	{"ferrari", "феррари"},
	{"lamborghini", "ламборгини", "ламборджини"},
}

// index хранит обратный маппинг: слово → индекс группы в synonymGroups.
var index map[string]int

func init() {
	index = make(map[string]int, len(synonymGroups)*4)
	for i, group := range synonymGroups {
		for _, word := range group {
			index[word] = i
		}
	}
}

// Expand возвращает все синонимы для данного запроса.
// Если запрос не найден в словаре, возвращает nil.
// Результат не включает сам запрос.
func Expand(query string) []string {
	lower := strings.ToLower(strings.TrimSpace(query))
	groupIdx, ok := index[lower]
	if !ok {
		return nil
	}

	group := synonymGroups[groupIdx]
	result := make([]string, 0, len(group)-1)
	for _, word := range group {
		if word != lower {
			result = append(result, word)
		}
	}
	return result
}

// ExpandAll ищет синонимы для каждого слова в запросе и возвращает все найденные.
// Для многословных запросов проверяет и полную фразу, и отдельные слова.
func ExpandAll(query string) []string {
	lower := strings.ToLower(strings.TrimSpace(query))

	var result []string
	seen := make(map[string]struct{})

	// Сначала проверяем полный запрос как единую фразу
	if synonyms := Expand(lower); len(synonyms) > 0 {
		for _, s := range synonyms {
			if _, ok := seen[s]; !ok {
				seen[s] = struct{}{}
				result = append(result, s)
			}
		}
	}

	// Затем проверяем отдельные слова
	words := strings.Fields(lower)
	if len(words) > 1 {
		for _, word := range words {
			if synonyms := Expand(word); len(synonyms) > 0 {
				for _, s := range synonyms {
					if _, ok := seen[s]; !ok {
						seen[s] = struct{}{}
						result = append(result, s)
					}
				}
			}
		}
	}

	return result
}
