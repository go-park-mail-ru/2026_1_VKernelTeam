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

	// "github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/app/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/pkg/validator"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/services/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage"
	"github.com/golang-jwt/jwt/v5"
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

// IsAdminRequest представляет собой структуру для запроса проверки прав администратора.
type IsAdminRequest struct {
	UserID int64 `json:"user_id"`
}

// IsAdminResponse представляет собой структуру для ответа на запрос проверки прав администратора.
type IsAdminResponse struct {
	IsAdmin bool `json:"is_admin"`
}

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

	app.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", port), // слушаем на всех интерфейсах
		Handler:      app.router,               // используем наш маршрутизатор
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
	a.router.HandleFunc("POST "+apiPrefix+"/auth/logout", a.handleLogout)

	// Защищенная ручка (оборачиваем в Middleware)
	// authMW := middleware.AuthMiddleware(a.blacklist, a.secret)
	// a.router.Handle("POST /auth/is-admin", authMW(http.HandlerFunc(a.handleIsAdmin)))

	// настройка раздачи статики
	fs := http.FileServer(http.Dir("static"))
	// StripPrefix убирает "/static/" из пути, чтобы искать сразу в папке static
	a.router.Handle("/static/", http.StripPrefix("/static/", fs))

	// регистрируем обработчик объявлений
	a.router.HandleFunc("GET "+apiPrefix+"/ads", a.handleGetAds)
}

// handleRegister обрабатывает запросы на регистрацию новых пользователей.
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
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
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

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	// извлекаем токен из куки
	cookie, err := r.Cookie("token")
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, "no token to logout")
		return
	}
	tokenString := cookie.Value

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok { // проверяем, что алгоритм подписи ожидаемый
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"]) // защита от атак с подменой алгоритма
		}
		return []byte(a.secret), nil
	})
	if err != nil { // если токен не распарсился, считаем его недействительным
		a.log.Error("failed to parse token", slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok { // если клеймы не в виде словаря, считаем токен недействительным
		a.log.Error("invalid token claims", slog.Any("claims", token.Claims))
		responser.RespondWithError(w, http.StatusBadRequest, "invalid token claims")
		return
	}

	jtiRaw, ok := claims["jti"].(string)
	if !ok { // если jti нет или он не строка, считаем токен недействительным
		a.log.Error("invalid or missing jti claim", slog.Any("claims", claims))
		responser.RespondWithError(w, http.StatusBadRequest, "invalid or missing jti claim")
		return
	}

	expRaw, ok := claims["exp"].(float64)
	if !ok { // если exp нет или он не число, считаем токен недействительным
		a.log.Error("invalid or missing exp claim", slog.Any("claims", claims))
		responser.RespondWithError(w, http.StatusBadRequest, "invalid or missing exp claim")
		return
	}

	jti := jtiRaw
	exp := time.Unix(int64(expRaw), 0)

	if err := a.services.Auth.Logout(r.Context(), jti, exp); err != nil {
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
