package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	entity "github.com/kofuk/premises/common/entity/web"
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

func (u *webAuthnUser) registerCredential(creds ...model.Credential) {
	for _, cred := range creds {
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
		c.Status(http.StatusBadRequest)
		return
	}

	challenge, err := protocol.CreateChallenge()
	if err != nil {
		log.WithError(err).Error("Error creating challenge")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}

	assertion := &protocol.CredentialAssertion{
		Response: protocol.PublicKeyCredentialRequestOptions{
			Challenge:          challenge,
			Timeout:            int(h.webauthn.Config.Timeouts.Login.Timeout.Milliseconds()),
			RelyingPartyID:     h.webauthn.Config.RPID,
			UserVerification:   h.webauthn.Config.AuthenticatorSelection.UserVerification,
			AllowedCredentials: make([]protocol.CredentialDescriptor, 0),
		},
	}

	session := sessions.Default(c)
	session.Set("hwkey_challenge", base64.RawURLEncoding.EncodeToString(challenge))
	session.Save()

	c.JSON(http.StatusOK, entity.SuccessfulResponse[*protocol.CredentialAssertion]{
		Success: true,
		Data:    assertion,
	})
}

func (h *Handler) handleLoginHardwarekeyFinish(c *gin.Context) {
	if c.GetHeader("Origin") != h.cfg.ControlPanel.Origin {
		c.Status(http.StatusBadGateway)
		return
	}

	session := sessions.Default(c)
	challenge, ok := session.Get("hwkey_challenge").(string)
	if !ok {
		log.Error("Client have no challenge")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
		return
	}
	session.Delete("hwkey_challenge")
	defer session.Save()

	parsedResponse, err := protocol.ParseCredentialRequestResponse(c.Request)
	if err != nil {
		log.WithError(err).Error("Error parsing credential request response")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
		return
	}

	// TODO: Improve this logic.

	userId := binary.BigEndian.Uint64(parsedResponse.Response.UserHandle)

	user := model.User{}
	if err := h.db.WithContext(c.Request.Context()).Preload("Credentials").Find(&user, userId).Error; err != nil {
		log.WithError(err).Error("User not found")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
		})
		return
	}

	waUser := webAuthnUser{
		user: user,
	}
	waUser.registerCredential(user.Credentials...)
	var allowedCredentials [][]byte
	for _, cred := range user.Credentials {
		allowedCredentials = append(allowedCredentials, cred.CredentialID)
	}

	sessionData := webauthn.SessionData{
		Challenge:            challenge,
		UserID:               waUser.WebAuthnID(),
		AllowedCredentialIDs: allowedCredentials,
		UserVerification:     h.webauthn.Config.AuthenticatorSelection.UserVerification,
	}

	cred, err := h.webauthn.ValidateLogin(&waUser, sessionData, parsedResponse)
	if err != nil {
		log.WithError(err).Error("error validating login")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasskeyVerify,
		})
		return
	}

	if cred.Authenticator.CloneWarning {
		log.Error("maybe a cloned authenticator used")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasskeyVerify,
		})
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
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}

	usedCred.AuthenticatorSignCount = cred.Authenticator.SignCount
	if err := h.db.WithContext(c.Request.Context()).Save(usedCred).Error; err != nil {
		log.WithError(err).Warn("failed to save credential")
	}

	session.Set("user_id", uint(userId))

	c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) setupWebauthnLoginRoutes(group *gin.RouterGroup) {
	group.POST("/begin", h.handleLoginHardwarekeyBegin)
	group.POST("/finish", h.handleLoginHardwarekeyFinish)
}
