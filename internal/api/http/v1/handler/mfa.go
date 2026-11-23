package handler

import (
	"WatchTower/internal/service/mfa"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MFAService exposes the complete authentication flow without a password-only shortcut.
type MFAService interface {
	Enroll(context.Context, string, string, string) error
	BeginLogin(context.Context, string, string) (string, error)
	Verify(context.Context, string, string) (mfa.AuthResult, error)
	Recover(context.Context, string) error
	Reset(context.Context, string, string, string) error
	ChangePassword(context.Context, string, string, string) error
}

func mfaBody(c *gin.Context, value any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return false
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return false
	}
	return true
}

func mfaError(c *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "authentication unavailable"
	switch {
	case errors.Is(err, mfa.ErrPolicy):
		status, message = 400, "invalid input or password policy"
	case errors.Is(err, mfa.ErrInvalid):
		status, message = 401, "invalid credentials or secret"
	case errors.Is(err, mfa.ErrExpired):
		status, message = 410, "secret expired"
	case errors.Is(err, mfa.ErrConflict):
		status, message = 409, "account already exists"
	case errors.Is(err, mfa.ErrDelivery):
		status, message = 503, "delivery unavailable"
	}
	c.JSON(status, gin.H{"error": message})
}

type mfaHandler struct{ service MFAService }

func RegisterMFARoutes(router gin.IRouter, service MFAService) {
	r := router.Group("/auth/mfa")
	h := mfaHandler{service: service}
	r.POST("/register", h.register)
	r.POST("/login", h.login)
	r.POST("/verify", h.verify)
	r.POST("/recover", h.recover)
	r.POST("/reset", h.reset)
	r.POST("/password", h.password)
}

func (h mfaHandler) register(c *gin.Context) {
	var request struct{ Login, Email, Password string }
	if !mfaBody(c, &request) {
		return
	}
	if err := h.service.Enroll(c.Request.Context(), request.Login, request.Email, request.Password); err != nil {
		mfaError(c, err)
		return
	}
	c.JSON(201, gin.H{"login": request.Login})
}

func (h mfaHandler) login(c *gin.Context) {
	var request struct{ Login, Password string }
	if !mfaBody(c, &request) {
		return
	}
	if request.Login == "" || request.Password == "" {
		mfaError(c, mfa.ErrPolicy)
		return
	}
	challenge, err := h.service.BeginLogin(c.Request.Context(), request.Login, request.Password)
	if err != nil {
		mfaError(c, err)
		return
	}
	c.JSON(202, gin.H{"challenge_id": challenge})
}

func (h mfaHandler) verify(c *gin.Context) {
	var request struct {
		ChallengeID string `json:"challenge_id"`
		Code        string `json:"code"`
	}
	if !mfaBody(c, &request) {
		return
	}
	result, err := h.service.Verify(c.Request.Context(), request.ChallengeID, request.Code)
	if err != nil {
		mfaError(c, err)
		return
	}
	c.JSON(200, result)
}

func (h mfaHandler) recover(c *gin.Context) {
	var request struct{ Login string }
	if !mfaBody(c, &request) {
		return
	}
	if request.Login == "" {
		mfaError(c, mfa.ErrPolicy)
		return
	}
	if err := h.service.Recover(c.Request.Context(), request.Login); err != nil {
		mfaError(c, err)
		return
	}
	c.JSON(202, gin.H{"message": "If the account exists, recovery instructions were sent"})
}

func (h mfaHandler) reset(c *gin.Context) {
	var request struct {
		Login       string `json:"login"`
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if !mfaBody(c, &request) {
		return
	}
	if err := h.service.Reset(c.Request.Context(), request.Login, request.Token, request.NewPassword); err != nil {
		mfaError(c, err)
		return
	}
	c.Status(204)
}

func (h mfaHandler) password(c *gin.Context) {
	var request struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !mfaBody(c, &request) {
		return
	}
	scheme, token, found := strings.Cut(c.GetHeader("Authorization"), " ")
	if !found || scheme != "Bearer" || token == "" {
		mfaError(c, mfa.ErrInvalid)
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), token, request.CurrentPassword, request.NewPassword); err != nil {
		mfaError(c, err)
		return
	}
	c.Status(204)
}
