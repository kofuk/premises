package handler

import (
	"net/http"
	"strings"

	entity "github.com/kofuk/premises/common/entity/web"
	"github.com/kofuk/premises/controlpanel/model"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) handleLogin(c echo.Context) error {
	if c.Request().Header.Get("Origin") != h.cfg.ControlPanel.Origin {
		return c.String(http.StatusBadGateway, "")
	}

	var cred entity.PasswordCredential
	if err := c.Bind(&cred); err != nil {
		log.WithError(err).Error("Failed to bind data")
		return c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	user := model.User{}
	if err := h.db.NewSelect().Model(&user).Column("id", "password", "initialized").Where("name = ? AND deleted_at IS NULL", cred.UserName).Scan(c.Request().Context()); err != nil {
		return c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
		})
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(cred.Password)) != nil {
		return c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
		})
	}

	session, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	if !user.Initialized {
		session.Values["change_password_user_id"] = user.ID
		session.Save(c.Request(), c.Response())

		return c.JSON(http.StatusOK, entity.SuccessfulResponse[entity.SessionState]{
			Success: true,
			Data: entity.SessionState{
				NeedsChangePassword: true,
			},
		})
	}

	session.Values["user_id"] = user.ID
	session.Save(c.Request(), c.Response())

	return c.JSON(http.StatusOK, entity.SuccessfulResponse[entity.SessionState]{
		Success: true,
		Data: entity.SessionState{
			NeedsChangePassword: false,
		},
	})
}

func (h *Handler) handleLogout(c echo.Context) error {
	session, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}
	delete(session.Values, "user_id")
	session.Save(c.Request(), c.Response())

	return c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func isAllowedPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	if strings.IndexAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz") < 0 {
		return false
	}
	if strings.IndexAny(password, "0123456789") < 0 {
		return false
	}
	return true
}

func (h *Handler) handleLoginResetPassword(c echo.Context) error {
	if c.Request().Header.Get("Origin") != h.cfg.ControlPanel.Origin {
		return c.String(http.StatusBadGateway, "")
	}

	session, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	userIdVal, ok := session.Values["change_password_user_id"]
	if !ok {
		return c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	userId, ok := userIdVal.(uint)
	if !ok {
		return c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	password := c.Request().PostFormValue("password")

	if !isAllowedPassword(password) {
		return c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasswordRule,
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("error registering user")
		return c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	if _, err := h.db.NewUpdate().Model((*model.User)(nil)).Set("password = ?", string(hashedPassword)).Set("initialized = ?", true).Where("id = ? AND deleted_at IS NULL", userId).Exec(c.Request().Context()); err != nil {
		log.WithError(err).Error("error updating password")
		return c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	session.Values["user_id"] = userId
	delete(session.Values, "change_password_user_id")
	session.Save(c.Request(), c.Response())

	return c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) setupRootRoutes(group *echo.Group) {
	group.POST("/login", h.handleLogin)
	group.POST("/logout", h.handleLogout)
	group.POST("/login/reset-password", h.handleLoginResetPassword)
}
