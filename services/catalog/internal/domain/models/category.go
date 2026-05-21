package models

//go:generate easyjson -all $GOFILE

// Category описывает категорию объявлений каталога.
type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
