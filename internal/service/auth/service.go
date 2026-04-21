package auth_service

import (
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type userContextKey struct{}

// AuthService defines the interface for the authentication authService.
type AuthService interface {
	Register(ctx context.Context, login, password string) error
	Login(ctx context.Context, login, password string) (string, error)
	ParseToken(ctx context.Context, tokenString string) (string, error)
}

type authService struct {
	userRepo  repo.UserRepository
	jwtSecret []byte
	tokenTTL  time.Duration
}

// NewService creates a new instance of the auth authService.
func NewService(userRepo repo.UserRepository, jwtSecret string, tokenTTL time.Duration) AuthService {
	return &authService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}
}

// Register creates a new user.
func (s *authService) Register(ctx context.Context, login, password string) error {
	// Check if user exists
	existingUser, err := s.userRepo.GetByLogin(ctx, login)
	if err == nil && existingUser != nil {
		return ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	usr := &user.User{
		Login:        login,
		PasswordHash: string(hashedPassword),
	}

	return s.userRepo.Create(ctx, usr)
}

// Login authenticates a user and returns a JWT token.
func (s *authService) Login(ctx context.Context, login, password string) (string, error) {
	usr, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		// user not found or error, return generic credentials error
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usr.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": usr.Login,
		"exp": time.Now().Add(s.tokenTTL).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ParseToken validates a JWT token and returns the login (subject).
func (s *authService) ParseToken(ctx context.Context, tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return "", ErrTokenInvalid
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if login, ok := claims["sub"].(string); ok {
			return login, nil
		}
	}

	return "", ErrTokenInvalid
}

// ContextWithUser adds user login to context
func ContextWithUser(ctx context.Context, login string) context.Context {
	return context.WithValue(ctx, userContextKey{}, login)
}

// UserFromContext retrieves user login from context
func UserFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value(userContextKey{})
	if login, ok := val.(string); ok {
		return login, true
	}
	return "", false
}
