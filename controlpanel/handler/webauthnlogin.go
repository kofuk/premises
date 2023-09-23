package handler

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net/http"

	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/kofuk/premises/controlpanel/model"
	log "github.com/sirupsen/logrus"
)

type webAuthnUser struct {
	user        model.User
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte {
	result := make([]byte, 8)
	binary.BigEndian.PutUint64(result, uint64(u.user.ID))
	return result
}

func (u *webAuthnUser) WebAuthnName() string {
	return u.user.Name
}

func (u *webAuthnUser) WebAuthnDisplayName() string {
	return u.user.Name
}

func (u *webAuthnUser) WebAuthnIcon() string {
	return ""
}

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (u *webAuthnUser) registerCredential(cred model.Credential) {
	u.credentials = append(u.credentials, webauthn.Credential{
		ID:              cred.CredentialID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		Authenticator: webauthn.Authenticator{
			AAGUID:    cred.AuthenticatorAAGUID,
			SignCount: cred.AuthenticatorSignCount,
		},
	})
}

func (u *webAuthnUser) getCredentialExcludeList() []protocol.CredentialDescriptor {
	var result []protocol.CredentialDescriptor
	for _, c := range u.credentials {
		result = append(result, protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: c.ID,
		})
	}
	return result
}

func (h *Handler) handleLoginHardwarekeyBegin(c *gin.Context) {
	if c.GetHeader("Origin") != h.cfg.ControlPanel.Origin {
		c.Status(http.StatusBadGateway)
		return
	}

	username := c.PostForm("username")

	user := model.User{}
	if err := h.db.WithContext(c.Request.Context()).Where("name = ?", username).Preload("Credentials").First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": h.L(h.cfg.ControlPanel.Locale, "login.error")})
		return
	}

	waUser := webAuthnUser{
		user: user,
	}
	if len(user.Credentials) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": h.L(h.cfg.ControlPanel.Locale, "login.error")})
		return
	}
	for _, c := range user.Credentials {
		waUser.registerCredential(c)
	}

	options, sessionData, err := h.webauthn.BeginLogin(&waUser)
	if err != nil {
		log.WithError(err).Error("error beginning login")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	marshaled, err := json.Marshal(sessionData)
	if err != nil {
		log.WithError(err).Error("error beginning login")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	session := sessions.Default(c)
	session.Set("hwkey_auth_user_id", user.ID)
	session.Set("hwkey_authentication", string(marshaled))
	session.Save()

	c.JSON(http.StatusOK, gin.H{"success": true, "options": options})
}

func (h *Handler) handleLoginHardwarekeyFinish(c *gin.Context) {
	if c.GetHeader("Origin") != h.cfg.ControlPanel.Origin {
		c.Status(http.StatusBadGateway)
		return
	}

	session := sessions.Default(c)
	userID := session.Get("hwkey_auth_user_id")
	marshaledData := session.Get("hwkey_authentication")
	session.Delete("hwkey_authentication")
	session.Delete("hwkey_auth_user_id")
	defer session.Save()

	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(marshaledData.(string)), &sessionData); err != nil {
		log.WithError(err).Error("Failed to unmarshal session data")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	user := model.User{}
	if err := h.db.WithContext(c.Request.Context()).Preload("Credentials").Find(&user, userID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": h.L(h.cfg.ControlPanel.Locale, "login.error")})
		return
	}

	waUser := webAuthnUser{
		user: user,
	}
	for _, c := range user.Credentials {
		waUser.registerCredential(c)
	}

	cred, err := h.webauthn.FinishLogin(&waUser, sessionData, c.Request)
	if err != nil {
		log.WithError(err).Error("error finishing login")
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": h.L(h.cfg.ControlPanel.Locale, "login.error")})
		return
	}

	if cred.Authenticator.CloneWarning {
		log.Error("maybe a cloned authenticator used")
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": h.L(h.cfg.ControlPanel.Locale, "login.error")})
		return
	}

	var usedCred *model.Credential
	for _, c := range user.Credentials {
		if bytes.Equal(c.CredentialID, cred.ID) {
			usedCred = &c
			break
		}
	}
	if usedCred == nil {
		log.WithError(err).Error("credential to update did not found")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	usedCred.AuthenticatorSignCount = cred.Authenticator.SignCount
	if err := h.db.WithContext(c.Request.Context()).Save(usedCred).Error; err != nil {
		log.WithError(err).Warn("failed to save credential")
	}

	session.Set("user_id", userID)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) setupWebauthnLoginRoutes(group *gin.RouterGroup) {
	group.POST("/begin", h.handleLoginHardwarekeyBegin)
	group.POST("/finish", h.handleLoginHardwarekeyFinish)
}
