package services

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	AuthRevision int64  `json:"auth_revision"`
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password minimal 8 karakter")
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func CheckPassword(password string, passwordHash string) error {
	return bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)
}

func GenerateJWT(
	userID string,
	email string,
	role string,
	revision ...int64,
) (string, error) {
	secret := os.Getenv("JWT_SECRET")

	if secret == "" {
		return "", errors.New("JWT_SECRET belum dikonfigurasi")
	}

	now := time.Now()
	var version int64
	if len(revision) > 0 {
		version = revision[0]
	}

	claims := Claims{
		UserID:       userID,
		AuthRevision: version,
		Email:        email,
		Role:         role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(
				now.Add(8 * time.Hour),
			),
			Issuer: "pt-fas",
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString([]byte(secret))
}

func ValidateJWT(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")

	if secret == "" {
		return nil, errors.New("JWT_SECRET belum dikonfigurasi")
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("metode token tidak valid")
			}

			return []byte(secret), nil
		},
		jwt.WithIssuer("pt-fas"),
		jwt.WithValidMethods([]string{
			jwt.SigningMethodHS256.Alg(),
		}),
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("token tidak valid")
	}

	return claims, nil
}
