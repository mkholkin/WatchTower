// Package mfa implements persistent, two-factor authentication.
package mfa

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	authsvc "WatchTower/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalid  = errors.New("invalid credentials or secret")
	ErrLocked   = errors.New("account locked")
	ErrExpired  = errors.New("secret expired")
	ErrPolicy   = errors.New("invalid input or password policy")
	ErrConflict = errors.New("account already exists")
	ErrDelivery = errors.New("delivery unavailable")
)

const issuer = "watchtower-mfa"
const accessScope = "access"
const changeScope = "password_change"

// Config controls token, challenge, password and attempt limits.
type Config struct {
	JWTSecret                              string
	TokenTTL, ChallengeTTL, PasswordMaxAge time.Duration
	MaxAttempts                            int
}

type Sender interface {
	Send(context.Context, string, string, string) error
}

type Service struct {
	pool   *pgxpool.Pool
	sender Sender
	cfg    Config
}

var _ authsvc.AuthService = (*Service)(nil)

type AuthResult struct {
	AccessToken            string `json:"access_token"`
	PasswordChangeRequired bool   `json:"password_change_required"`
}

type tokenClaims struct {
	jwt.RegisteredClaims
	Scope   string `json:"scope"`
	Version int64  `json:"version"`
	MFA     bool   `json:"mfa"`
}

type account struct {
	login, email, password    string
	attempts                  int
	locked                    bool
	changed                   time.Time
	version                   int64
	otp, recovery             []byte
	otpExpiry, recoveryExpiry time.Time
}

func New(pool *pgxpool.Pool, sender Sender, cfg Config) (*Service, error) {
	if pool == nil || sender == nil || len(cfg.JWTSecret) < 32 {
		return nil, errors.New("MFA requires database, sender and JWT secret of at least 32 bytes")
	}
	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = time.Hour
	}
	if cfg.ChallengeTTL <= 0 {
		cfg.ChallengeTTL = 5 * time.Minute
	}
	if cfg.PasswordMaxAge <= 0 {
		cfg.PasswordMaxAge = 90 * 24 * time.Hour
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	return &Service{pool: pool, sender: sender, cfg: cfg}, nil
}

func validPassword(password string) bool { return len(password) > 0 && len(password) <= 72 }

func (s *Service) Enroll(ctx context.Context, login, email, password string) error {
	address, err := mail.ParseAddress(email)
	if !validPassword(password) || login == "" || strings.TrimSpace(login) != login || len(login) > 255 || strings.ContainsAny(login, "\r\n") || err != nil || address.Address != email || strings.ContainsAny(email, "\r\n") {
		return ErrPolicy
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.transaction(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO "user" (login,password_hash) VALUES ($1,$2)`, login, string(hash)); err != nil {
			var pgerr *pgconn.PgError
			if errors.As(err, &pgerr) && pgerr.Code == "23505" {
				return ErrConflict
			}
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO auth_mfa (login,email) VALUES ($1,$2)`, login, email)
		return err
	})
}

// transaction commits credential failure counters as well as successful operations.
// Delivery and database errors roll back all pending state.
func (s *Service) transaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	result := fn(tx)
	if result != nil && !errors.Is(result, ErrInvalid) && !errors.Is(result, ErrLocked) && !errors.Is(result, ErrExpired) && !errors.Is(result, ErrPolicy) {
		return result
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return result
}

func load(ctx context.Context, tx pgx.Tx, byChallenge bool, key string) (account, error) {
	var a account
	predicate := "m.login=$1"
	if byChallenge {
		predicate = "m.challenge_id=$1"
	}
	err := tx.QueryRow(ctx, `SELECT m.login,m.email,u.password_hash,m.failed_attempts,m.locked,m.password_changed_at,m.token_version,
 COALESCE(m.otp_hash,''::bytea),COALESCE(m.otp_expires_at,'epoch'::timestamptz),
 COALESCE(m.recovery_hash,''::bytea),COALESCE(m.recovery_expires_at,'epoch'::timestamptz)
 FROM auth_mfa m JOIN "user" u ON u.login=m.login WHERE `+predicate+` FOR UPDATE OF m,u`, key).Scan(&a.login, &a.email, &a.password, &a.attempts, &a.locked, &a.changed, &a.version, &a.otp, &a.otpExpiry, &a.recovery, &a.recoveryExpiry)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrInvalid
	}
	return a, err
}

func (s *Service) failure(ctx context.Context, tx pgx.Tx, a account) error {
	locked := a.attempts+1 >= s.cfg.MaxAttempts
	_, err := tx.Exec(ctx, `UPDATE auth_mfa SET failed_attempts=failed_attempts+1,locked=$2,
 token_version=token_version+CASE WHEN $2 THEN 1 ELSE 0 END,
 challenge_id=CASE WHEN $2 THEN NULL ELSE challenge_id END,
 otp_hash=CASE WHEN $2 THEN NULL ELSE otp_hash END,
 otp_expires_at=CASE WHEN $2 THEN NULL ELSE otp_expires_at END WHERE login=$1`, a.login, locked)
	if err != nil {
		return err
	}
	if locked {
		return ErrLocked
	}
	return ErrInvalid
}

func secretHash(secret string) []byte { sum := sha256.Sum256([]byte(secret)); return sum[:] }
func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) BeginLogin(ctx context.Context, login, password string) (string, error) {
	var challenge string
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		a, err := load(ctx, tx, false, login)
		if err != nil {
			return err
		}
		if a.locked {
			return ErrLocked
		}
		if !validPassword(password) || bcrypt.CompareHashAndPassword([]byte(a.password), []byte(password)) != nil {
			return s.failure(ctx, tx, a)
		}
		challenge, err = randomSecret()
		if err != nil {
			return err
		}
		n, err := rand.Int(rand.Reader, big.NewInt(1000000))
		if err != nil {
			return err
		}
		code := fmt.Sprintf("%06d", n.Int64())
		_, err = tx.Exec(ctx, `UPDATE auth_mfa SET challenge_id=$2,otp_hash=$3,otp_expires_at=$4 WHERE login=$1`, a.login, challenge, secretHash(code), time.Now().Add(s.cfg.ChallengeTTL))
		if err != nil {
			return err
		}
		if err = s.sender.Send(ctx, a.email, "login", code); err != nil {
			return ErrDelivery
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return challenge, nil
}

func (s *Service) Verify(ctx context.Context, challengeID, code string) (AuthResult, error) {
	var result AuthResult
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		a, err := load(ctx, tx, true, challengeID)
		if err != nil {
			return err
		}
		if a.locked {
			return ErrLocked
		}
		if !time.Now().Before(a.otpExpiry) {
			return ErrExpired
		}
		if len(code) != 6 || subtle.ConstantTimeCompare(secretHash(code), a.otp) != 1 {
			return s.failure(ctx, tx, a)
		}
		_, err = tx.Exec(ctx, `UPDATE auth_mfa SET failed_attempts=0,challenge_id=NULL,otp_hash=NULL,otp_expires_at=NULL WHERE login=$1`, a.login)
		if err != nil {
			return err
		}
		result.PasswordChangeRequired = !time.Now().Before(a.changed.Add(s.cfg.PasswordMaxAge))
		scope := accessScope
		if result.PasswordChangeRequired {
			scope = changeScope
		}
		now := time.Now()
		claims := tokenClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, Subject: a.login, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.TokenTTL))}, Scope: scope, Version: a.version, MFA: true}
		result.AccessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
		return err
	})
	if err != nil {
		return AuthResult{}, err
	}
	return result, nil
}

func (s *Service) Recover(ctx context.Context, login string) error {
	return s.transaction(ctx, func(tx pgx.Tx) error {
		a, err := load(ctx, tx, false, login)
		if errors.Is(err, ErrInvalid) {
			return nil
		}
		if err != nil {
			return err
		}
		secret, err := randomSecret()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE auth_mfa SET recovery_hash=$2,recovery_expires_at=$3 WHERE login=$1`, a.login, secretHash(secret), time.Now().Add(10*time.Minute))
		if err != nil {
			return err
		}
		if err = s.sender.Send(ctx, a.email, "recovery", secret); err != nil {
			return ErrDelivery
		}
		return nil
	})
}

func (s *Service) Reset(ctx context.Context, login, token, newPassword string) error {
	if !validPassword(newPassword) {
		return ErrPolicy
	}
	return s.transaction(ctx, func(tx pgx.Tx) error {
		a, err := load(ctx, tx, false, login)
		if err != nil {
			return err
		}
		if len(a.recovery) == 0 || subtle.ConstantTimeCompare(secretHash(token), a.recovery) != 1 {
			return ErrInvalid
		}
		if !time.Now().Before(a.recoveryExpiry) {
			return ErrExpired
		}
		if bcrypt.CompareHashAndPassword([]byte(a.password), []byte(newPassword)) == nil {
			return ErrPolicy
		}
		return replacePassword(ctx, tx, a.login, newPassword)
	})
}

func replacePassword(ctx context.Context, tx pgx.Tx, login, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE "user" SET password_hash=$2 WHERE login=$1`, login, string(hash))
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE auth_mfa SET failed_attempts=0,locked=false,password_changed_at=clock_timestamp(),token_version=token_version+1,
 challenge_id=NULL,otp_hash=NULL,otp_expires_at=NULL,recovery_hash=NULL,recovery_expires_at=NULL WHERE login=$1`, login)
	return err
}

func (s *Service) parse(token string) (*tokenClaims, error) {
	claims := &tokenClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) { return []byte(s.cfg.JWTSecret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(issuer), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || claims.Subject == "" || claims.IssuedAt == nil || !claims.MFA || claims.Version < 1 || (claims.Scope != accessScope && claims.Scope != changeScope) {
		return nil, ErrInvalid
	}
	return claims, nil
}

func (s *Service) ChangePassword(ctx context.Context, bearer, currentPassword, newPassword string) error {
	if !validPassword(newPassword) || newPassword == currentPassword {
		return ErrPolicy
	}
	claims, err := s.parse(bearer)
	if err != nil {
		return ErrInvalid
	}
	return s.transaction(ctx, func(tx pgx.Tx) error {
		a, err := load(ctx, tx, false, claims.Subject)
		if err != nil {
			return err
		}
		if a.locked {
			return ErrLocked
		}
		if claims.Version != a.version {
			return ErrInvalid
		}
		if !validPassword(currentPassword) || bcrypt.CompareHashAndPassword([]byte(a.password), []byte(currentPassword)) != nil {
			return s.failure(ctx, tx, a)
		}
		return replacePassword(ctx, tx, a.login, newPassword)
	})
}

func (s *Service) Register(context.Context, string, string) error {
	return authsvc.ErrInvalidCredentials
}
func (s *Service) Login(context.Context, string, string) (string, error) {
	return "", authsvc.ErrInvalidCredentials
}

func (s *Service) ParseToken(token string) (string, error) {
	claims, err := s.parse(token)
	if err != nil || claims.Scope != accessScope {
		return "", authsvc.ErrTokenInvalid
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var locked bool
	var version int64
	var changed time.Time
	err = s.pool.QueryRow(ctx, `SELECT locked,token_version,password_changed_at FROM auth_mfa WHERE login=$1`, claims.Subject).Scan(&locked, &version, &changed)
	if err != nil || locked || version != claims.Version || !time.Now().Before(changed.Add(s.cfg.PasswordMaxAge)) {
		return "", authsvc.ErrTokenInvalid
	}
	return claims.Subject, nil
}
