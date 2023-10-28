package handler

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/kofuk/premises/controlpanel/entity"
	"github.com/kofuk/premises/controlpanel/model"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	robotsTxt = `User-agent: *
Disallow: /
`
)

func (h *Handler) L(locale, msgId string) string {
	localizer := i18n.NewLocalizer(h.i18nData, locale)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: msgId})
	if err != nil {
		log.WithError(err).Error("Error loading localized message. Fallback to \"en\"")
		localizer := i18n.NewLocalizer(h.i18nData, "en")
		msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: msgId})
		if err != nil {
			log.WithError(err).Error("Error loading localized message (fallback)")
			return msgId
		}
		return msg
	}
	return msg
}

func (h *Handler) handleLogin(c *gin.Context) {
	if c.GetHeader("Origin") != h.cfg.ControlPanel.Origin {
		c.Status(http.StatusBadGateway)
		return
	}

	var cred entity.PasswordCredential
	if err := c.BindJSON(&cred); err != nil {
		log.WithError(err).Error("Failed to bind data")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
			Reason:    "Bad request",
		})
		return
	}

	user := model.User{}
	if err := h.db.WithContext(c.Request.Context()).Where("name = ?", cred.UserName).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
			Reason:    h.L(h.cfg.ControlPanel.Locale, "login.error"),
		})
		return
	}

	session := sessions.Default(c)
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(cred.Password)) == nil {
		if !user.Initialized {
			session.Set("change_password_user_id", user.ID)
			session.Save()

			c.JSON(http.StatusOK, entity.SuccessfulResponse[entity.SessionState]{
				Success: true,
				Data: entity.SessionState{
					NeedsChangePassword: true,
				},
			})
		} else {
			session.Set("user_id", user.ID)
			session.Save()

			c.JSON(http.StatusOK, entity.SuccessfulResponse[entity.SessionState]{
				Success: true,
				Data: entity.SessionState{
					NeedsChangePassword: false,
				},
			})
		}
	} else {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
			Reason:    h.L(h.cfg.ControlPanel.Locale, "login.error"),
		})
	}
}

func (h *Handler) handleLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("user_id")
	session.Save()

	c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
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

func (h *Handler) handleLoginResetPassword(c *gin.Context) {
	if c.GetHeader("Origin") != h.cfg.ControlPanel.Origin {
		c.Status(http.StatusBadGateway)
		return
	}

	session := sessions.Default(c)
	user_id := session.Get("change_password_user_id")

	username := c.PostForm("username")
	password := c.PostForm("password")

	user := model.User{}
	if err := h.db.WithContext(c.Request.Context()).Where("name = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
			Reason:    h.L(h.cfg.ControlPanel.Locale, "login.error"),
		})
		return
	}

	if user.ID != user_id {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
			Reason:    h.L(h.cfg.ControlPanel.Locale, "login.error"),
		})
		return
	}

	if !isAllowedPassword(password) {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasswordRule,
			Reason:    "Disallowed password",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("error registering user")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
			Reason:    "Error registering user",
		})
		return
	}
	user.Password = string(hashedPassword)
	user.Initialized = true

	if err := h.db.WithContext(c.Request.Context()).Save(user).Error; err != nil {
		log.WithError(err).Error("error updating password")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
			Reason:    "Error registering user",
		})
		return
	}

	session.Set("user_id", user.ID)
	session.Delete("change_password_user_id")
	session.Save()

	c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleRobotsTxt(c *gin.Context) {
	c.Writer.Write([]byte(robotsTxt))
}

func (h *Handler) setupRootRoutes(group *gin.RouterGroup) {
	group.POST("/login", h.handleLogin)
	group.POST("/logout", h.handleLogout)
	group.POST("/login/reset-password", h.handleLoginResetPassword)
}
