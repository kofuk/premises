package handler

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	entity "github.com/kofuk/premises/common/entity/web"
	"github.com/kofuk/premises/controlpanel/model"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

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
		})
		return
	}

	user := model.User{}
	if err := h.db.NewSelect().Model(&user).Column("id", "password", "initialized").Where("name = ? AND deleted_at IS NULL", cred.UserName).Scan(c.Request.Context()); err != nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
		})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(cred.Password)) != nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
		})
		return
	}

	session := sessions.Default(c)

	if !user.Initialized {
		session.Set("change_password_user_id", user.ID)
		session.Save()

		c.JSON(http.StatusOK, entity.SuccessfulResponse[entity.SessionState]{
			Success: true,
			Data: entity.SessionState{
				NeedsChangePassword: true,
			},
		})
		return
	}

	session.Set("user_id", user.ID)
	session.Save()

	c.JSON(http.StatusOK, entity.SuccessfulResponse[entity.SessionState]{
		Success: true,
		Data: entity.SessionState{
			NeedsChangePassword: false,
		},
	})
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

	password := c.PostForm("password")

	if !isAllowedPassword(password) {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasswordRule,
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("error registering user")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}

	if _, err := h.db.NewUpdate().Model((*model.User)(nil)).Set("password = ?", string(hashedPassword)).Set("initialized = ?", true).Where("id = ? AND deleted_at IS NULL", user_id).Exec(c.Request.Context()); err != nil {
		log.WithError(err).Error("error updating password")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}

	session.Set("user_id", user_id)
	session.Delete("change_password_user_id")
	session.Save()

	c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) setupRootRoutes(group *gin.RouterGroup) {
	group.POST("/login", h.handleLogin)
	group.POST("/logout", h.handleLogout)
	group.POST("/login/reset-password", h.handleLoginResetPassword)
}
