// Пакет httpapp реализует HTTP-интерфейс поверх сервиса auth. Он
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

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/services/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/utils"

)

// App представляет HTTP-приложение с маршрутизатором, логгером и
// ссылкой на сервис аутентификации.
type App struct {
	log    *slog.Logger
	router *http.ServeMux
	port   int
	srv    *http.Server
	auth   Auth
}

// Auth описывает минимальный набор методов сервиса аутентификации, который
// использует HTTP-приложение.
type Auth interface {
	Login(ctx context.Context, email string, password string, appID int64) (token string, err error)
	RegisterNewUser(ctx context.Context, email string, password string) (userID int64, err error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	UserID int64 `json:"user_id"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	AppID    int64  `json:"app_id"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type IsAdminRequest struct {
	UserID int64 `json:"user_id"`
}

type IsAdminResponse struct {
	IsAdmin bool `json:"is_admin"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// New создаёт новый HTTP-сервер с заданной конфигурацией и сервисом auth.
func New(
	log *slog.Logger,
	authService Auth,
	port int,
) *App {
	app := &App{
		log:    log,
		router: http.NewServeMux(),
		port:   port,
		auth:   authService,
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
	a.router.HandleFunc("POST /register", a.handleRegister)
	a.router.HandleFunc("POST /login", a.handleLogin)
	a.router.HandleFunc("POST /is-admin", a.handleIsAdmin)
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

	if req.AppID == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "app_id is required")
		return
	}

	token, err := a.auth.Login(r.Context(), req.Email, req.Password, req.AppID)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			utils.RespondWithError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}

		a.log.Error("failed to login", slog.String("error", err.Error()))
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, LoginResponse{Token: token})
}

// handleIsAdmin проверяет, является ли указанный пользователь администратором.
func (a *App) handleIsAdmin(w http.ResponseWriter, r *http.Request) {
	var req IsAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	isAdmin, err := a.auth.IsAdmin(r.Context(), req.UserID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "user not found")
			return
		}

		a.log.Error("failed to check admin status", slog.String("error", err.Error()))
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to check admin status")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, IsAdminResponse{IsAdmin: isAdmin})
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
