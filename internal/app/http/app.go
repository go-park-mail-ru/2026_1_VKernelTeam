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
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/services/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage/blacklist"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

// App представляет HTTP-приложение с маршрутизатором, логгером и
// ссылкой на сервис аутентификации.
type App struct {
	log       *slog.Logger
	router    *http.ServeMux
	port      int
	srv       *http.Server
	auth      Auth
	blacklist *blacklist.InMemory
	secret    string
}

// Auth описывает минимальный набор методов сервиса аутентификации, который
// использует HTTP-приложение.
type Auth interface {
	Login(ctx context.Context, email string, password string) (token string, err error)
	RegisterNewUser(ctx context.Context, email string, password string) (userID int64, err error)
	// IsAdmin(ctx context.Context, userID int64) (bool, error)
	Logout(ctx context.Context, jti string, exp time.Time) error
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
	authService Auth,
	bl *blacklist.InMemory,
	port int,
	secret string,
) *App {
	app := &App{
		log:       log,
		router:    http.NewServeMux(),
		port:      port,
		auth:      authService,
		blacklist: bl,
		secret:    secret,
	}

	app.setupRoutes()

	app.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: app.router,
	}

	return app
}

// setupRoutes регистрирует HTTP-обработчики.
func (a *App) setupRoutes() {
	a.router.HandleFunc("POST /auth/register", a.handleRegister)
	a.router.HandleFunc("POST /auth/login", a.handleLogin)
	a.router.HandleFunc("POST /auth/logout", http.HandlerFunc(a.handleLogout))

	// Защищенная ручка (оборачиваем в Middleware)
	// authMW := middleware.AuthMiddleware(a.blacklist, a.secret)
	// a.router.Handle("POST /auth/is-admin", authMW(http.HandlerFunc(a.handleIsAdmin)))
}

// handleRegister обрабатывает запросы на регистрацию новых пользователей.
func (a *App) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "email is required")
		return
	}

	if req.Password == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "password is required")
		return
	}

	userID, err := a.auth.RegisterNewUser(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			utils.RespondWithError(w, http.StatusConflict, "user already exists")
			return
		}

		a.log.Error("failed to register user", slog.String("error", err.Error()))
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, RegisterResponse{UserID: userID})
}

// handleLogin обрабатывает запросы на вход в систему и возвращает JWT.
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "email is required")
		return
	}

	if req.Password == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "password is required")
		return
	}

	token, err := a.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			utils.RespondWithError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}

		a.log.Error("failed to login", slog.String("error", err.Error()))
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,                     // JS не увидит куку
		Secure:   true,                     // передача только по HTTPS
		Path:     "/",                      // доступна везде
		SameSite: http.SameSiteLaxMode,     // защита от CSRF атак
		MaxAge:   int(time.Hour.Seconds()), // время жизни - час
	})

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
		utils.RespondWithError(w, http.StatusBadRequest, "no token to logout")
		return
	}
	tokenString := cookie.Value

	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		a.log.Error("failed to get token", slog.String("error", err.Error()))
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get token")
		return
	}

	claims := token.Claims.(jwt.MapClaims)

	jti := claims["jti"].(string)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)

	if err := a.auth.Logout(r.Context(), jti, exp); err != nil {
		a.log.Error("failed to logout in service", slog.String("error", err.Error()))
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to logout")
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

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// MustRun запускает сервер и паникует при любой ошибке.
func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

// Run запускает HTTP-сервер и возвращает ошибку при сбое.
func (a *App) Run() error {
	const op = "httpapp.Run"

	a.log.Info("http server started", slog.String("addr", a.srv.Addr))

	if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Stop корректно останавливает сервер.
func (a *App) Stop() {
	const op = "httpapp.Stop"

	a.log.With(slog.String("op", op)).
		Info("stopping http server", slog.Int("port", a.port))

	a.srv.Close()
}
