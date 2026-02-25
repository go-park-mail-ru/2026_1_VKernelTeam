package main

import (
	"fmt"
	"net/http"
)

func main() {
	// регистрируем обработчик
	http.HandleFunc("/ads", getAdsHandler)

	fmt.Println("Server started at :8080")

	// запускаем http-сервер на порту 8080
	err := http.ListenAndServe(":8080", nil)

	// если порта занят, возникнет ошибка
	if err != nil {
		panic("Server failed to start: " + err.Error())
	}
}
