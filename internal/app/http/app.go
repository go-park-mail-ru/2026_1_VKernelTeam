// Package httpapp реализует HTTP-интерфейс поверх сервиса auth. Он
// предоставляет обработчики для маршрутов регистрации, входа и проверки
// прав администратора, а также простую структуру сервера.
package httpapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/app/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/pkg/validator"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/services/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/storage"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/storage/blacklist"

	_ "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	opRun  = "httpapp.Run"
	opStop = "httpapp.Stop"
)

// App представляет HTTP-приложение с маршрутизатором, логгером и
// ссылкой на сервис аутентификации.
type App struct {
	log       *slog.Logger
	router    *http.ServeMux
	port      int
	srv       *http.Server
	services  Services
	blacklist storage.TokenRevoker
	tokenTTL  time.Duration
	secret    string
}

// Ads описывает методы сервиса объявлений.
type Ads interface {
	GetAll() []models.Ad
}

// Auth описывает минимальный набор методов сервиса аутентификации, который
// использует HTTP-приложение.
type Auth interface {
	Login(ctx context.Context, email string, password string) (token string, err error)
	RegisterNewUser(ctx context.Context, email string, password string) (userID int64, err error)
	// IsAdmin(ctx context.Context, userID int64) (bool, error)
	Logout(ctx context.Context, jti string, exp time.Time) error
}

// Services объединяет все бизнес-сервисы приложения.
type Services struct {
	Ads  Ads
	Auth Auth
}

// RegisterRequest представляет собой структуру для запроса на регистрацию пользователя.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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
	Token string `json:"token"`
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

// New создаёт новый HTTP-сервер с заданной конфигурацией и сервисом auth.
func New(
	log *slog.Logger,
	services Services,
	bl storage.TokenRevoker,
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

func (a *App) setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,                      // JS не увидит куку
		Secure:   true,                      // передача только по HTTPS
		Path:     "/",                       // доступна везде
		SameSite: http.SameSiteLaxMode,      // защита от CSRF атак
		MaxAge:   int(a.tokenTTL.Seconds()), // время жизни
	})
}

// setupRoutes регистрирует HTTP-обработчики.
func (a *App) setupRoutes() {
	a.router.HandleFunc("POST "+apiPrefix+"/auth/register", a.handleRegister)
	a.router.HandleFunc("POST "+apiPrefix+"/auth/login", a.handleLogin)

	// Защищенная ручка (оборачиваем в Middleware)
	authMW := middleware.AuthMiddleware(a.log, a.blacklist.(*blacklist.InMemory), a.secret)
	a.router.Handle("POST "+apiPrefix+"/auth/logout", authMW(http.HandlerFunc(a.handleLogout)))

	// Ручка для Swagger UI
	// Она будет доступна по адресу /swagger/index.html
	a.router.Handle("/swagger/", httpSwagger.WrapHandler)

	// настройка раздачи статики
	fs := http.FileServer(http.Dir("static"))
	// StripPrefix убирает "/static/" из пути, чтобы искать сразу в папке static
	a.router.Handle("/static/", http.StripPrefix("/static/", fs))

	// регистрируем обработчик объявлений
	a.router.HandleFunc("GET "+apiPrefix+"/ads", a.handleGetAds)
}

// handleRegister обрабатывает запросы на регистрацию новых пользователей.
// @Summary Регистрация
// @Tags auth
// @Accept json
// @Produce json
// @Param input body RegisterRequest true "Данные регистрации"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} ErrorResponse
// @Router /auth/register [post]
func (a *App) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validator.ValidateEmail(req.Email); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validator.ValidatePassword(req.Password); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := a.services.Auth.RegisterNewUser(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			responser.RespondWithError(w, http.StatusConflict, "user already exists")
			return
		}

		a.log.Error("failed to register user", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	a.log.Info(
		"user registered successfully",
		slog.Int64("user_id", userID),
		slog.String("email", req.Email),
	)

	// сразу логиним
	token, err := a.services.Auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		a.log.Error("auto-login failed after registration", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, "registered, but failed to login")
		return
	}

	a.setAuthCookie(w, token)

	responser.RespondWithJSON(w, http.StatusCreated, RegisterResponse{UserID: userID})
}

// handleLogin обрабатывает запросы на вход в систему и возвращает JWT.
// @Summary Вход
// @Tags auth
// @Accept json
// @Produce json
// @Param input body LoginRequest true "Данные входа"
// @Success 200 {object} map[string]string
// @Failure 401 {object} ErrorResponse
// @Router /auth/login [post]
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validator.ValidateEmail(req.Email); err != nil {
		a.log.Warn(
			"invalid login attempt",
			slog.String("email", req.Email),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validator.ValidatePassword(req.Password); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := a.services.Auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			responser.RespondWithError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}

		a.log.Error("failed to login", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	a.setAuthCookie(w, token)

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleIsAdmin проверяет, является ли указанный пользователь администратором.
// TODO: доставать токен из куки или бурать ручку совсем
// func (a *App) handleIsAdmin(w http.ResponseWriter, r *http.Request) {
// 	var req IsAdminRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
// 		return
// 	}

// 	if req.UserID == 0 {
// 		utils.RespondWithError(w, http.StatusBadRequest, "user_id is required")
// 		return
// 	}

// 	isAdmin, err := a.auth.IsAdmin(r.Context(), req.UserID)
// 	if err != nil {
// 		if errors.Is(err, storage.ErrUserNotFound) {
// 			utils.RespondWithError(w, http.StatusNotFound, "user not found")
// 			return
// 		}

// 		a.log.Error("failed to check admin status", slog.String("error", err.Error()))
// 		utils.RespondWithError(w, http.StatusInternalServerError, "failed to check admin status")
// 		return
// 	}

// 	utils.RespondWithJSON(w, http.StatusOK, IsAdminResponse{IsAdmin: isAdmin})
// }

// handleLogout обрабатывает запросы на выход из системы, добавляя jti токена в черный список.
// @Summary Выход
// @Tags auth
// @Success 200 {object} map[string]string
// @Router /auth/logout [post]
func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	a.log.Info("logout attempt", slog.String("op", "handleLogout"))

	// достаём jti из контекста
	jti, ok := r.Context().Value(middleware.JtiKey).(string)
	if !ok {
		a.log.Error("jti not found in context")
		responser.RespondWithError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := a.services.Auth.Logout(r.Context(), jti, time.Now().Add(a.tokenTTL)); err != nil {
		a.log.Error("failed to logout in service", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, "failed to logout")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,              // удаляем куку
		Expires:  time.Unix(0, 0), // на всякий случай делаем просроченной
	})

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleGetAds обрабатывает запросы на получение списка объявлений.
// @Summary Список объявлений
// @Tags ads
// @Produce json
// @Success 200 {array} models.Ad
// @Router /ads [get]
func (a *App) handleGetAds(w http.ResponseWriter, r *http.Request) {
	// обрабатываем только GET запросы
	if r.Method != http.MethodGet {
		// формируем и отправляем ошибку 405
		responser.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// копируем список объявлений без гонки данных
	adsList := a.services.Ads.GetAll()

	// формируем и отправляем ответ
	responser.RespondWithJSON(w, http.StatusOK, adsList)
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

	a.srv.Close()
}
