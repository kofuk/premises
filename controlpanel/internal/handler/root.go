package handler

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/boj/redistore/v2"
	"github.com/gorilla/sessions"
	"github.com/kofuk/premises/controlpanel/internal/auth"
	"github.com/kofuk/premises/controlpanel/internal/db/model"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/web"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) handleLogin(c *echo.Context) error {
	var cred web.PasswordCredential
	if err := c.Bind(&cred); err != nil {
		slog.Error("Failed to bind data", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	user := model.User{}
	if err := h.db.NewSelect().Model(&user).Column("id", "password", "initialized").Where("name = ? AND deleted_at IS NULL", cred.UserName).Scan(c.Request().Context()); err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
		})
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(cred.Password)) != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
		})
	}

	session, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	if !user.Initialized {
		session.Values["change_password_user_id"] = user.ID
		session.Save(c.Request(), c.Response())

		return c.JSON(http.StatusOK, web.SuccessfulResponse[web.SessionState]{
			Success: true,
			Data: web.SessionState{
				NeedsChangePassword: true,
			},
		})
	}

	token, err := h.authService.CreateToken(c.Request().Context(), user.ID, []auth.Scope{auth.ScopeAdmin})
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	session.Values["access_token"] = token.Token
	session.Save(c.Request(), c.Response())

	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.SessionState]{
		Success: true,
		Data: web.SessionState{
			NeedsChangePassword: false,
		},
	})
}

func (h *Handler) handleLogout(c *echo.Context) error {
	session, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	accessToken := session.Values["access_token"].(string)

	if err := h.authService.RevokeToken(c.Request().Context(), accessToken); err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	delete(session.Values, "access_token")
	session.Save(c.Request(), c.Response())

	return c.JSON(http.StatusOK, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func isAllowedPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	if !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz") {
		return false
	}
	if !strings.ContainsAny(password, "0123456789") {
		return false
	}
	return true
}

func (h *Handler) handleLoginResetPassword(c *echo.Context) error {
	session, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	userIdVal, ok := session.Values["change_password_user_id"]
	if !ok {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	userId, ok := userIdVal.(uint)
	if !ok {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	password := c.Request().PostFormValue("password")

	if !isAllowedPassword(password) {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasswordRule,
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("error registering user", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	if _, err := h.db.NewUpdate().Model((*model.User)(nil)).Set("password = ?", string(hashedPassword)).Set("initialized = ?", true).Where("id = ? AND deleted_at IS NULL", userId).Exec(c.Request().Context()); err != nil {
		slog.Error("error updating password", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	session.Values["user_id"] = userId
	delete(session.Values, "change_password_user_id")
	session.Save(c.Request(), c.Response())

	return c.JSON(http.StatusOK, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleSessionData(c *echo.Context) error {
	session, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	sessionData := web.SessionData{}

	accessToken, ok := session.Values["access_token"].(string)
	if !ok {
		sessionData.LoggedIn = false
	} else {
		token, err := h.authService.Get(c.Request().Context(), accessToken)
		if err == nil {
			sessionData.LoggedIn = true
			sessionData.AccessToken = token.Token
		}
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.SessionData]{
		Success: true,
		Data:    sessionData,
	})
}

func (h *Handler) handleHealth(c *echo.Context) error {
	_, err := h.redis.Ping(c.Request().Context()).Result()
	if err != nil {
		slog.Error("Can't connect to Redis", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "error")
	}

	row, err := h.db.QueryContext(c.Request().Context(), "SELECT 1")
	if err != nil {
		slog.Error("Can't connect to PostgreSQL", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "error")
	}
	row.Close()

	return c.String(http.StatusOK, "ok")
}

func (h *Handler) setupRootRoutes(group *echo.Group) {
	store, err := redistore.NewStore(
		[][]byte{[]byte(h.cfg.Secret)},
		redistore.WithAddress("tcp", h.cfg.RedisAddress),
		redistore.WithAuth(h.cfg.RedisUser, h.cfg.RedisPassword),
	)
	if err != nil {
		slog.Error("Failed to initialize Redis session store", slog.Any("error", err))
		os.Exit(1)
	}
	store.Options = &sessions.Options{
		MaxAge:   60 * 60 * 24 * 30,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	store.SetKeyPrefix("session:")

	group.Use(session.Middleware(store))
	group.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Verify that the request is sent from allowed origin (if needed).
			if c.Request().Method == http.MethodPost {
				if c.Request().Header.Get("Origin") != h.cfg.Origin {
					slog.Error("origin not allowed", slog.String("origin", c.Request().Header.Get("Origin")))
					return c.JSON(http.StatusOK, web.ErrorResponse{
						Success:   false,
						ErrorCode: entity.ErrBadRequest,
					})
				}
			}

			return next(c)
		}
	})
	group.POST("/api/internal/login", h.handleLogin)
	group.POST("/api/internal/logout", h.handleLogout)
	group.GET("/api/internal/session-data", h.handleSessionData)
	group.POST("/api/internal/login/reset-password", h.handleLoginResetPassword)
	group.GET("/health", h.handleHealth)
}
