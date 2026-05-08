package handler

import (
	apigen "WatchTower/internal/api/http/v1/gen"
	authsvc "WatchTower/internal/service/auth"
	"strings"

	"github.com/gin-gonic/gin"
)

func (a *ApiHandler) AuthStrictMiddleware(next apigen.StrictHandlerFunc, operationID string) apigen.StrictHandlerFunc {
	return func(c *gin.Context, req any) (any, error) {
		if operationID == "LoginUser" || operationID == "RegisterUser" {
			return next(c, req)
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			return next(c, req)
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return next(c, req)
		}

		login, err := a.authSvc.ParseToken(parts[1])
		if err != nil {
			return next(c, req)
		}

		c.Set(authsvc.UserContextKey, login)

		return next(c, req)
	}
}
