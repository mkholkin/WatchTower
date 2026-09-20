package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	"WatchTower/internal/domain/repo"
	authsvc "WatchTower/internal/service/auth"
)

func (c *Client) Register(ctx context.Context, login, password string) error {
	response, err := c.api.RegisterUserWithResponse(ctx, apiv2gen.RegisterUserJSONRequestBody{Login: login, Password: password})
	if err != nil {
		return fmt.Errorf("register user: %w", err)
	}
	if response.StatusCode() == http.StatusCreated {
		return nil
	}
	err = responseError(response.StatusCode(), response.Body)
	if errors.Is(err, repo.ErrAlreadyExists) {
		return authsvc.ErrUserAlreadyExists
	}
	return err
}

func (c *Client) Login(ctx context.Context, login, password string) (string, error) {
	response, err := c.api.LoginUserWithResponse(ctx, apiv2gen.LoginUserJSONRequestBody{Login: login, Password: password})
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		if response.StatusCode() == http.StatusUnauthorized {
			return "", authsvc.ErrInvalidCredentials
		}
		return "", responseError(response.StatusCode(), response.Body)
	}

	c.token = response.JSON200.AccessToken
	c.login = login
	return c.token, nil
}

func (c *Client) ParseToken(token string) (string, error) {
	if token == "" || token != c.token || c.login == "" {
		return "", authsvc.ErrTokenInvalid
	}
	return c.login, nil
}
