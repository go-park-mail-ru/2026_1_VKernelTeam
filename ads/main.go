package main

import (
	"fmt"
	"net/http"
)

func main() {
	// настройка раздачи статики
	fs := http.FileServer(http.Dir("../static"))
	// StripPrefix убирает "/static/" из пути, чтобы искать сразу в папке static
	http.Handle("/static/", http.StripPrefix("/static/", fs))

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
