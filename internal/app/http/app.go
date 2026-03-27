// Package httpapp реализует HTTP-интерфейс поверх сервиса auth. Он
// предоставляет обработчики для маршрутов регистрации, входа и проверки
// прав администратора, а также простую структуру сервера.
package httpapp

//go:generate mockgen -source=app.go -destination=mocks/mock_app.go

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	api "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/api"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/delivery/handlers"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"

	_ "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/api"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	opRun  = "httpapp.Run"
	opStop = "httpapp.Stop"
)

// Auth описывает минимальный набор методов сервиса аутентификации
type Auth interface {
	Login(ctx context.Context, email string, password string) (string, string, models.User, error)
	ValidateTokenAndGetUser(ctx context.Context, tokenString string) (models.User, error)
	RegisterNewUser(ctx context.Context, email string, password string, name string) (userID int64, err error)
	Logout(ctx context.Context, jti string, exp time.Time, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
}

// Ads описывает методы сервиса объявлений
type Ads interface {
	GetAllAds(ctx context.Context) ([]models.Ad, error)
	CreateAd(ctx context.Context, req *dto.CreateAdRequest) (int64, error)
}

// TokenChecker интерфейс для проверки отозванных токенов
type TokenChecker interface {
	Check(jti string) bool
}

// Services объединяет все бизнес-сервисы приложения
type Services struct {
	Ads  Ads
	Auth Auth
}

// App представляет HTTP-приложение с маршрутизатором, логгером и
// ссылкой на сервис аутентификации.
type App struct {
	log          *slog.Logger
	router       *http.ServeMux
	port         int
	srv          *http.Server
	services     Services
	blacklist    TokenChecker
	tokenTTL     time.Duration
	secret       string
	authHandlers *handlers.AuthHandlers
	adsHandlers  *handlers.AdsHandlers
}

// New создаёт новый HTTP-сервер с заданной конфигурацией и сервисом auth.
func New(
	log *slog.Logger,
	services Services,
	bl TokenChecker,
	port int,
	tokenTTL time.Duration,
	refreshTTL time.Duration,
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

	app.authHandlers = handlers.NewAuthHandlers(log, handlers.Services{
		Auth: services.Auth,
		Ads:  services.Ads,
	}, tokenTTL, refreshTTL, secret)
	app.adsHandlers = handlers.NewAdsHandlers(log, handlers.Services{
		Auth: services.Auth,
		Ads:  services.Ads,
	}, tokenTTL)

	app.setupRoutes()

	handlerWithCSRF := middleware.CSRFMiddleware(app.router)
	finalHandler := middleware.CORSMiddleware(handlerWithCSRF)

	app.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", port), // слушаем на всех интерфейсах
		Handler:      finalHandler,             // передаем итоговую цепочку
		ReadTimeout:  15 * time.Second,         // ограничиваем время чтения запроса
		WriteTimeout: 15 * time.Second,         // ограничиваем время записи ответа
		IdleTimeout:  60 * time.Second,         // время жизни соединения
	}

	return app
}

// setupRoutes регистрирует HTTP-обработчики.
func (a *App) setupRoutes() {
	a.router.HandleFunc("POST "+api.ApiPrefix+"/auth/register", a.authHandlers.HandleRegister)
	a.router.HandleFunc("POST "+api.ApiPrefix+"/auth/login", a.authHandlers.HandleLogin)
	a.router.HandleFunc("POST "+api.ApiPrefix+"/auth/refresh", a.authHandlers.HandleRefresh)

	// Защищенная ручка (оборачиваем в Middleware)
	authMW := middleware.AuthMiddleware(a.log, a.blacklist, a.secret)
	a.router.Handle("POST "+api.ApiPrefix+"/auth/logout", authMW(http.HandlerFunc(a.authHandlers.HandleLogout)))

	// Ручка для Swagger UI
	// Она будет доступна по адресу /swagger/index.html
	a.router.Handle("/swagger/", httpSwagger.WrapHandler)

	// настройка раздачи статики
	fs := http.FileServer(http.Dir("static"))
	// StripPrefix убирает "/static/" из пути, чтобы искать сразу в папке static
	a.router.Handle("/static/", http.StripPrefix("/static/", fs))

	// регистрируем обработчик объявлений
	a.router.HandleFunc("GET "+api.ApiPrefix+"/ads", a.adsHandlers.HandleGetAds)
	a.router.Handle("POST "+api.ApiPrefix+"/ads", authMW(http.HandlerFunc(a.adsHandlers.HandleCreateAd)))
	// a.router.HandleFunc("GET "+api.ApiPrefix+"/ads/{id}", a.adsHandlers.HandleGetAdByID)
	// a.router.HandleFunc("PUT "+api.ApiPrefix+"/ads/{id}", a.adsHandlers.HandleUpdateAdByID)
	// a.router.HandleFunc("DELETE "+api.ApiPrefix+"/ads/{id}", a.adsHandlers.HandleDeleteAd)
	// a.router.HandleFunc("POST "+api.ApiPrefix+"/ads/{id}/close", a.adsHandlers.HandleCloseAdByID)
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
