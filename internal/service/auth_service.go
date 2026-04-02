package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
)

const AuthTokenTTL = 24 * time.Hour

type authTokenClaims struct {
	Subject    string `json:"sub"`
	Username   string `json:"username"`
	Permission string `json:"permission"`
	ExpiresAt  int64  `json:"exp"`
}

type AuthService struct {
	db         database.Database
	authSecret []byte
	now        func() time.Time
}

func NewAuthService(db database.Database, authSecret string) *AuthService {
	return &AuthService{
		db:         db,
		authSecret: []byte(authSecret),
		now:        time.Now,
	}
}

func (s *AuthService) SyncBootstrapAdmin(ctx context.Context, username string, password string) (entity.UserEntity, error) {
	_ = ctx
	hash, err := hashPassword(password)
	if err != nil {
		return entity.UserEntity{}, err
	}
	user, err := s.db.UpsertUser(entity.UserEntity{
		Username:     strings.TrimSpace(username),
		PasswordHash: hash,
		Permission:   entity.UserPermissionAdmin,
	})
	if err != nil {
		return entity.UserEntity{}, err
	}
	return user, nil
}

func (s *AuthService) Authenticate(ctx context.Context, username string, password string) (dto.LoginResponse, error) {
	_ = ctx
	user, err := s.db.GetUserByUsername(strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.LoginResponse{}, ErrInvalidCredentials
		}
		return dto.LoginResponse{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return dto.LoginResponse{}, ErrInvalidCredentials
	}

	token, err := s.signUserToken(user)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	return dto.LoginResponse{
		Token: token,
		User:  toAuthUser(user),
	}, nil
}

func (s *AuthService) GetUserFromToken(token string) (entity.UserEntity, error) {
	claims, err := s.parseToken(token)
	if err != nil {
		return entity.UserEntity{}, ErrUnauthorized
	}

	user, err := s.db.GetUserByID(claims.Subject)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.UserEntity{}, ErrUnauthorized
		}
		return entity.UserEntity{}, err
	}
	if user.Username != claims.Username || user.Permission != claims.Permission {
		return entity.UserEntity{}, ErrUnauthorized
	}
	return user, nil
}

func (s *AuthService) signUserToken(user entity.UserEntity) (string, error) {
	headerPayload := map[string]string{
		"alg": "HS256",
		"typ": "KineticJWT",
	}
	headerJSON, err := json.Marshal(headerPayload)
	if err != nil {
		return "", err
	}

	claimsJSON, err := json.Marshal(authTokenClaims{
		Subject:    user.ID,
		Username:   user.Username,
		Permission: user.Permission,
		ExpiresAt:  s.now().Add(AuthTokenTTL).Unix(),
	})
	if err != nil {
		return "", err
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := header + "." + payload
	signature := s.sign(unsigned)
	return unsigned + "." + signature, nil
}

func (s *AuthService) parseToken(raw string) (authTokenClaims, error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 3 {
		return authTokenClaims{}, ErrUnauthorized
	}

	unsigned := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(s.sign(unsigned))) {
		return authTokenClaims{}, ErrUnauthorized
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return authTokenClaims{}, ErrUnauthorized
	}

	var claims authTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return authTokenClaims{}, ErrUnauthorized
	}
	if claims.Subject == "" || claims.Username == "" || claims.Permission == "" {
		return authTokenClaims{}, ErrUnauthorized
	}
	if s.now().Unix() >= claims.ExpiresAt {
		return authTokenClaims{}, ErrUnauthorized
	}

	return claims, nil
}

func (s *AuthService) sign(value string) string {
	mac := hmac.New(sha256.New, s.authSecret)
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

type UserService struct {
	db database.Database
}

func NewUserService(db database.Database) *UserService {
	return &UserService{db: db}
}

func (s *UserService) ListUsers() ([]entity.UserEntity, error) {
	return s.db.ListUsers()
}

func (s *UserService) CreateUser(username string, password string) (entity.UserEntity, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return entity.UserEntity{}, fmt.Errorf("username and password are required")
	}
	if _, err := s.db.GetUserByUsername(username); err == nil {
		return entity.UserEntity{}, ErrUserAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return entity.UserEntity{}, err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return entity.UserEntity{}, err
	}
	user, err := s.db.CreateUser(entity.UserEntity{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: hash,
		Permission:   entity.UserPermissionAdmin,
	})
	if err != nil {
		return entity.UserEntity{}, err
	}
	return user, nil
}

func (s *UserService) UpdatePassword(userID string, password string) error {
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("password is required")
	}
	if _, err := s.db.GetUserByID(userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	return s.db.UpdateUserPassword(userID, hash)
}

func (s *UserService) DeleteUser(userID string) error {
	if _, err := s.db.GetUserByID(userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	return s.db.DeleteUser(userID)
}

func toAuthUser(user entity.UserEntity) dto.AuthUser {
	return dto.AuthUser{
		ID:         user.ID,
		Username:   user.Username,
		Permission: user.Permission,
	}
}
