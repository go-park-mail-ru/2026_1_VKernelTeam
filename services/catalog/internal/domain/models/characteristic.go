package models

//go:generate easyjson -all $GOFILE

// CategoryCharacteristic описывает предопределённую характеристику категории.
type CategoryCharacteristic struct {
	ID            int64    `json:"id"`
	CategoryID    int64    `json:"category_id"`
	Name          string   `json:"name"`
	AllowedValues []string `json:"allowed_values"`
	SortOrder     int      `json:"sort_order"`
}

// ProductCharacteristic описывает значение категорийной характеристики объявления.
type ProductCharacteristic struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ProductCustomCharacteristic описывает пользовательскую характеристику объявления.
type ProductCustomCharacteristic struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
