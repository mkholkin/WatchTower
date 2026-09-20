package handler

import (
	"net/http"
	"strings"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	authsvc "WatchTower/internal/service/auth"

	"github.com/gin-gonic/gin"
)

func (a *APIHandler) AuthStrictMiddleware(next apiv2gen.StrictHandlerFunc, operationID string) apiv2gen.StrictHandlerFunc {
	return func(c *gin.Context, req any) (any, error) {
		if operationID == "LoginUser" || operationID == "RegisterUser" {
			return next(c, req)
		}

		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			abortUnauthorized(c, "a bearer token is required")
			return nil, nil
		}

		login, err := a.authSvc.ParseToken(parts[1])
		if err != nil {
			abortUnauthorized(c, "the bearer token is invalid")
			return nil, nil
		}

		c.Request = c.Request.WithContext(authsvc.ContextWithUser(c.Request.Context(), login))
		return next(c, req)
	}
}

func abortUnauthorized(c *gin.Context, message string) {
	c.Header("WWW-Authenticate", "Bearer")
	c.AbortWithStatusJSON(http.StatusUnauthorized, apiv2gen.Error{Code: "unauthorized", Message: message})
}
