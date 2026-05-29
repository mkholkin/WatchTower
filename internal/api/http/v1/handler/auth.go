package handler

import (
	v1 "WatchTower/internal/api/http/v1"
	apigen "WatchTower/internal/api/http/v1/gen"
	authsvc "WatchTower/internal/service/auth"
	"context"
	"errors"
)

var loginUserResponseFactory = v1.ResponseFactory[apigen.LoginUserResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.LoginUserResponseObject {
		return apigen.LoginUser400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.LoginUserResponseObject {
		return apigen.LoginUser401JSONResponse{apigen.N401JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.LoginUserResponseObject {
		return apigen.LoginUser500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var registerUserResponseFactory = v1.ResponseFactory[apigen.RegisterUserResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.RegisterUserResponseObject {
		return apigen.RegisterUser400JSONResponse{apigen.N400JSONResponse(er)}
	},
	409: func(er apigen.ErrorResponse) apigen.RegisterUserResponseObject {
		return apigen.RegisterUser409JSONResponse{apigen.N409JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.RegisterUserResponseObject {
		return apigen.RegisterUser500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

func authErrorResponse(code, message string) apigen.ErrorResponse {
	return apigen.ErrorResponse{Code: code, Message: message}
}

func authResponseFromFactory[T any](factory v1.ResponseFactory[T], err error) T {
	switch {
	case errors.Is(err, authsvc.ErrInvalidCredentials):
		if f, ok := factory[401]; ok {
			return f(authErrorResponse("UNAUTHORIZED", err.Error()))
		}
	case errors.Is(err, authsvc.ErrUserAlreadyExists):
		if f, ok := factory[409]; ok {
			return f(authErrorResponse("ALREADY_EXISTS", err.Error()))
		}
	}
	return v1.ResponseFromFactory(factory, err)
}

func (a *ApiHandler) LoginUser(ctx context.Context, request apigen.LoginUserRequestObject) (apigen.LoginUserResponseObject, error) {
	if request.Body == nil {
		return loginUserResponseFactory[400](authErrorResponse("INVALID_DATA", "request body is required")), nil
	}

	accessToken, err := a.authSvc.Login(ctx, request.Body.Login, request.Body.Password)
	if err != nil {
		return authResponseFromFactory(loginUserResponseFactory, err), nil
	}

	response := apigen.LoginUser200JSONResponse{
		AccessToken: accessToken,
	}

	return response, nil
}

func (a *ApiHandler) RegisterUser(ctx context.Context, request apigen.RegisterUserRequestObject) (apigen.RegisterUserResponseObject, error) {
	if request.Body == nil {
		return registerUserResponseFactory[400](authErrorResponse("INVALID_DATA", "request body is required")), nil
	}

	if err := a.authSvc.Register(ctx, request.Body.Login, request.Body.Password); err != nil {
		return authResponseFromFactory(registerUserResponseFactory, err), nil
	}

	response := apigen.RegisterUser201JSONResponse{
		Login: request.Body.Login,
	}

	return response, nil
}
