package main

import (
	"ads/internal/ads"
	"fmt"
	"net/http"
	"time"
)

func main() {
	// настройка раздачи статики
	fs := http.FileServer(http.Dir("../static"))
	// StripPrefix убирает "/static/" из пути, чтобы искать сразу в папке static
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// регистрируем обработчик
	http.HandleFunc("/", ads.GetAdsHandler)

	server := &http.Server{
		Addr: ":8080",

		// используем http.DefaultServeMux
		Handler: nil,

		// ставим таймауты на чтение и запись, чтоыб соединение не висело вечно
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,

		// 2^20 байт = 1024^2 байт = 1 Мб - защита от больших заголовков
		MaxHeaderBytes: 1 << 20,
	}

	fmt.Println("Server started at :8080")

	// запускаем http-сервер на порту 8080
	err := server.ListenAndServe()

	// если порта занят, возникнет ошибка
	if err != nil {
		panic("Server failed to start: " + err.Error())
	}
}
