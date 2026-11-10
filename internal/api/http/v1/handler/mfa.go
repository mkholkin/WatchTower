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

func RegisterMFARoutes(router gin.IRouter, service MFAService) {
	r := router.Group("/auth/mfa")
	r.POST("/register", func(c *gin.Context) {
		var request struct{ Login, Email, Password string }
		if !mfaBody(c, &request) {
			return
		}
		if err := service.Enroll(c.Request.Context(), request.Login, request.Email, request.Password); err != nil {
			mfaError(c, err)
			return
		}
		c.JSON(201, gin.H{"login": request.Login})
	})
	r.POST("/login", func(c *gin.Context) {
		var request struct{ Login, Password string }
		if !mfaBody(c, &request) {
			return
		}
		if request.Login == "" || request.Password == "" {
			mfaError(c, mfa.ErrPolicy)
			return
		}
		challenge, err := service.BeginLogin(c.Request.Context(), request.Login, request.Password)
		if err != nil {
			mfaError(c, err)
			return
		}
		c.JSON(202, gin.H{"challenge_id": challenge})
	})
	r.POST("/verify", func(c *gin.Context) {
		var request struct {
			ChallengeID string `json:"challenge_id"`
			Code        string `json:"code"`
		}
		if !mfaBody(c, &request) {
			return
		}
		result, err := service.Verify(c.Request.Context(), request.ChallengeID, request.Code)
		if err != nil {
			mfaError(c, err)
			return
		}
		c.JSON(200, result)
	})
	r.POST("/recover", func(c *gin.Context) {
		var request struct{ Login string }
		if !mfaBody(c, &request) {
			return
		}
		if request.Login == "" {
			mfaError(c, mfa.ErrPolicy)
			return
		}
		if err := service.Recover(c.Request.Context(), request.Login); err != nil {
			mfaError(c, err)
			return
		}
		c.JSON(202, gin.H{"message": "If the account exists, recovery instructions were sent"})
	})
	r.POST("/reset", func(c *gin.Context) {
		var request struct {
			Login       string `json:"login"`
			Token       string `json:"token"`
			NewPassword string `json:"new_password"`
		}
		if !mfaBody(c, &request) {
			return
		}
		if err := service.Reset(c.Request.Context(), request.Login, request.Token, request.NewPassword); err != nil {
			mfaError(c, err)
			return
		}
		c.Status(204)
	})
	r.POST("/password", func(c *gin.Context) {
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
		if err := service.ChangePassword(c.Request.Context(), token, request.CurrentPassword, request.NewPassword); err != nil {
			mfaError(c, err)
			return
		}
		c.Status(204)
	})
}
