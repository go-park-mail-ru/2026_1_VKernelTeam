// Package httpapp реализует HTTP-интерфейс поверх сервиса auth. Он
// предоставляет обработчики для маршрутов регистрации, входа и проверки
// прав администратора, а также простую структуру сервера.
package httpapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/delivery/handlers"
	blacklist "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/blacklist"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"

	_ "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/api"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	opRun  = "httpapp.Run"
	opStop = "httpapp.Stop"
)

// ошибки HTTP-обработчиков
var (
	ErrInvalidRequestBody   = "invalid request body"
	ErrUserAlreadyExists    = "user already exists"
	ErrFailedToRegisterUser = "failed to register user"
	ErrAutoLoginFailed      = "registered, but failed to login"
	ErrFailedToLogin        = "failed to login"
	ErrInternalError        = "internal error"
	ErrFailedToLogout       = "failed to logout"
	ErrMethodNotAllowed     = "Method not allowed"
)

// App представляет HTTP-приложение с маршрутизатором, логгером и
// ссылкой на сервис аутентификации.
type App struct {
	log          *slog.Logger
	router       *http.ServeMux
	port         int
	srv          *http.Server
	services     handlers.Services
	blacklist    auth.TokenRevoker
	tokenTTL     time.Duration
	secret       string
	authHandlers *handlers.AuthHandlers
	adsHandlers  *handlers.AdsHandlers
}

// RegisterRequest представляет собой структуру для запроса на регистрацию пользователя.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// RegisterResponse представляет собой структуру для ответа на запрос регистрации пользователя.
type RegisterResponse struct {
	UserID int64 `json:"user_id"`
}

// LoginRequest представляет собой структуру для запроса на вход в систему, содержащую email и пароль.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse представляет собой структуру для ответа на запрос входа в систему, содержащую JWT-токен.
type LoginResponse struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

// // IsAdminRequest представляет собой структуру для запроса проверки прав администратора.
// type IsAdminRequest struct {
// 	UserID int64 `json:"user_id"`
// }

// // IsAdminResponse представляет собой структуру для ответа на запрос проверки прав администратора.
// type IsAdminResponse struct {
// 	IsAdmin bool `json:"is_admin"`
// }

// ErrorResponse представляет собой структуру для отправки ошибок в формате JSON.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidationErrors представляет собой структуру для отправки ошибок валидации по полям.
type ValidationErrors struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
}

// New создаёт новый HTTP-сервер с заданной конфигурацией и сервисом auth.
func New(
	log *slog.Logger,
	services handlers.Services,
	bl auth.TokenRevoker,
	port int,
	tokenTTL time.Duration,
	secret string,
) *App {
	app := &App{
		log:       log,
		router:    http.NewServeMux(),
		port:      port,
		services:  services,
		tokenTTL:  tokenTTL,
		blacklist: bl,
		secret:    secret,
	}

	app.authHandlers = handlers.NewAuthHandlers(log, services, bl, tokenTTL, secret)
	app.adsHandlers = handlers.NewAdsHandlers(log, services)

	app.setupRoutes()

	handlerWithCORS := middleware.CORSMiddleware(app.router)

	app.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", port), // слушаем на всех интерфейсах
		Handler:      handlerWithCORS,          // используем наш маршрутизатор с CORS
		ReadTimeout:  15 * time.Second,         // ограничиваем время чтения запроса
		WriteTimeout: 15 * time.Second,         // ограничиваем время записи ответа
		IdleTimeout:  60 * time.Second,         // время жизни соединения
	}

	return app
}

const apiPrefix = "/api/v1"

// setupRoutes регистрирует HTTP-обработчики.
func (a *App) setupRoutes() {
	a.router.HandleFunc("POST "+apiPrefix+"/auth/register", a.authHandlers.HandleRegister)
	a.router.HandleFunc("POST "+apiPrefix+"/auth/login", a.authHandlers.HandleLogin)

	// Защищенная ручка (оборачиваем в Middleware)
	authMW := middleware.AuthMiddleware(a.log, a.blacklist.(*blacklist.InMemory), a.secret)
	a.router.Handle("POST "+apiPrefix+"/auth/logout", authMW(http.HandlerFunc(a.authHandlers.HandleLogout)))

	// Ручка для Swagger UI
	// Она будет доступна по адресу /swagger/index.html
	a.router.Handle("/swagger/", httpSwagger.WrapHandler)

	// настройка раздачи статики
	fs := http.FileServer(http.Dir("static"))
	// StripPrefix убирает "/static/" из пути, чтобы искать сразу в папке static
	a.router.Handle("/static/", http.StripPrefix("/static/", fs))

	// регистрируем обработчик объявлений
	a.router.HandleFunc("GET "+apiPrefix+"/ads", a.adsHandlers.HandleGetAds)
}

// MustRun запускает сервер и паникует при любой ошибке.
func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

// Run запускает HTTP-сервер и возвращает ошибку при сбое.
func (a *App) Run() error {
	a.log.Info("http server started", slog.String("addr", a.srv.Addr))

	if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("%s: %w", opRun, err)
	}

	return nil
}

// Stop корректно останавливает сервер.
func (a *App) Stop() {
	a.log.With(slog.String("opStop", opStop)).
		Info("stopping http server", slog.Int("port", a.port))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.srv.Shutdown(ctx); err != nil {
		a.log.Error("failed to shutdown http server gracefully", slog.String("error", err.Error()))
	}
}
