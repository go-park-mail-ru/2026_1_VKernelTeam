// Package httpapp реализует HTTP-интерфейс поверх сервиса auth. Он
// предоставляет обработчики для маршрутов регистрации, входа и проверки
// прав администратора, а также простую структуру сервера.
package httpapp

//go:generate mockgen -source=app.go -destination=mocks/mock_app.go

import (
	"context"
	"fmt"
	"io"
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
	GetProfile(ctx context.Context, userID int64) (models.User, error)
	UpdateProfile(ctx context.Context, userID int64, name string) (models.User, error)
	UpdateAvatar(ctx context.Context, userID int64, file io.ReadSeeker, filename string) (models.User, error)
}

// Ads описывает методы сервиса объявлений
type Ads interface {
	GetAllAds(ctx context.Context) ([]models.Ad, error)
	GetAdByID(ctx context.Context, id int64) (models.Ad, error)
	CreateAd(ctx context.Context, req *dto.CreateAdRequest) (int64, error)
	UpdateAd(ctx context.Context, req *dto.UpdateAdRequest) error
	DeleteAd(ctx context.Context, id int64, userID int64) error
	CloseAd(ctx context.Context, id int64, userID int64) error
	GetAdsByUserID(ctx context.Context, userID int64) ([]models.Ad, error)
	AddFavorite(ctx context.Context, userID int64, adID int64) error
	RemoveFavorite(ctx context.Context, userID int64, adID int64) error
	GetUserFavorites(ctx context.Context, userID int64) ([]models.Ad, error)
}

type Cart interface {
	AddToCart(ctx context.Context, userID, productID int64) error
	RemoveFromCart(ctx context.Context, userID, productID int64) error
	GetCart(ctx context.Context, userID int64) (*dto.CartResponse, error)
	Checkout(ctx context.Context, userID int64) (*dto.CheckoutResponse, error)
}

// TokenChecker интерфейс для проверки отозванных токенов
type TokenChecker interface {
	Check(jti string) bool
}

// Services объединяет все бизнес-сервисы приложения
type Services struct {
	Ads  Ads
	Auth Auth
	Cart Cart
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
	cartHandlers *handlers.CartHandlers
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
		Cart: services.Cart,
	}, tokenTTL, refreshTTL, secret)
	app.adsHandlers = handlers.NewAdsHandlers(log, handlers.Services{
		Auth: services.Auth,
		Ads:  services.Ads,
		Cart: services.Cart,
	}, tokenTTL)
	app.cartHandlers = handlers.NewCartHandlers(log, handlers.Services{
		Auth: services.Auth,
		Ads:  services.Ads,
		Cart: services.Cart,
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
	prefix := api.ApiPrefix

	// Публичные ручки
	a.router.HandleFunc("POST "+prefix+"/auth/register", a.authHandlers.HandleRegister)
	a.router.HandleFunc("POST "+prefix+"/auth/login", a.authHandlers.HandleLogin)
	a.router.HandleFunc("POST "+prefix+"/auth/refresh", a.authHandlers.HandleRefresh)

	// Обработчии объявлений
	a.router.HandleFunc("GET "+prefix+"/ads", a.adsHandlers.HandleGetAds)
	a.router.HandleFunc("GET "+prefix+"/ads/{id}", a.adsHandlers.HandleGetAdByID)

	// Публичный профиль продавца и его объявления
	a.router.HandleFunc("GET "+prefix+"/users/{id}", a.authHandlers.HandleGetPublicProfile)
	a.router.HandleFunc("GET "+prefix+"/users/{id}/ads", a.adsHandlers.HandleGetUserAds)

	// Защищенные ручки (нужен JWT)
	authMW := middleware.AuthMiddleware(a.log, a.blacklist, a.secret)

	// Обработчики объявлений
	a.router.Handle("POST "+prefix+"/ads", authMW(http.HandlerFunc(a.adsHandlers.HandleCreateAd)))
	a.router.Handle("PUT "+prefix+"/ads/{id}", authMW(http.HandlerFunc(a.adsHandlers.HandleUpdateAdByID)))
	a.router.Handle("DELETE "+prefix+"/ads/{id}", authMW(http.HandlerFunc(a.adsHandlers.HandleDeleteAd)))
	a.router.Handle("POST "+prefix+"/ads/{id}/close", authMW(http.HandlerFunc(a.adsHandlers.HandleCloseAdByID)))

	// Корзина (защищено)
	a.router.Handle("GET "+prefix+"/cart", authMW(http.HandlerFunc(a.cartHandlers.HandleGetCart)))
	a.router.Handle("POST "+prefix+"/cart", authMW(http.HandlerFunc(a.cartHandlers.HandleAddToCart)))
	a.router.Handle("DELETE "+prefix+"/cart/{id}", authMW(http.HandlerFunc(a.cartHandlers.HandleRemoveFromCart)))
	a.router.Handle("POST "+prefix+"/cart/checkout", authMW(http.HandlerFunc(a.cartHandlers.HandleCheckout)))

	// Избранное
	a.router.Handle("POST "+prefix+"/ads/{id}/favorite", authMW(http.HandlerFunc(a.adsHandlers.HandleAddToFavorites)))
	a.router.Handle("DELETE "+prefix+"/ads/{id}/favorite", authMW(http.HandlerFunc(a.adsHandlers.HandleDeleteFromFavorites)))
	a.router.Handle("GET "+prefix+"/profile/favorites", authMW(http.HandlerFunc(a.adsHandlers.HandleGetFavorites)))

	// Выход
	a.router.Handle("POST "+prefix+"/auth/logout", authMW(http.HandlerFunc(a.authHandlers.HandleLogout)))

	// Личный профиль
	a.router.Handle("GET "+prefix+"/profile", authMW(http.HandlerFunc(a.authHandlers.HandleGetProfile)))
	a.router.Handle("PATCH "+prefix+"/profile", authMW(http.HandlerFunc(a.authHandlers.HandleUpdateProfile)))

	// Аватар
	a.router.Handle("POST "+prefix+"/profile/avatar", authMW(http.HandlerFunc(a.authHandlers.HandleUploadAvatar)))

	// Ручка для Swagger UI
	// Она будет доступна по адресу /swagger/index.html
	a.router.Handle("/swagger/", httpSwagger.WrapHandler)

	// настройка раздачи статики
	fs := http.FileServer(http.Dir("static"))
	// StripPrefix убирает "/static/" из пути, чтобы искать сразу в папке static
	a.router.Handle("/static/", http.StripPrefix("/static/", fs))
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
