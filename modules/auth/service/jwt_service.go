package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type JWTService interface {
	GenerateAccessToken(userId string, role string) string
	GenerateRefreshToken() (string, time.Time)
	ValidateToken(token string) (*jwt.Token, error)
	GetUserIDByToken(token string) (string, error)
}

type jwtCustomClaim struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type jwtService struct {
	secretKey     string
	issuer        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTService() JWTService {
	return &jwtService{
		secretKey:     getSecretKey(),
		issuer:        "teralur",
		accessExpiry:  time.Minute * 15,
		refreshExpiry: time.Hour * 24 * 7,
	}
}

func getSecretKey() string {
	secretKey := os.Getenv("JWT_SECRET")
	if len(secretKey) < 16 {
		// Gagal tertutup: tanpa secret yang layak, token bisa dipalsukan siapa pun.
		log.Fatal("JWT_SECRET wajib diisi (minimal 16 karakter)")
	}
	return secretKey
}

func (j *jwtService) GenerateAccessToken(userId string, role string) string {
	claims := jwtCustomClaim{
		userId,
		role,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.accessExpiry)),
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tx, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		log.Println(err)
	}
	return tx
}

func (j *jwtService) GenerateRefreshToken() (string, time.Time) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		log.Println(err)
		return "", time.Time{}
	}

	refreshToken := base64.StdEncoding.EncodeToString(b)
	expiresAt := time.Now().Add(j.refreshExpiry)

	return refreshToken, expiresAt
}

func (j *jwtService) parseToken(t_ *jwt.Token) (any, error) {
	if _, ok := t_.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("unexpected signing method %v", t_.Header["alg"])
	}
	return []byte(j.secretKey), nil
}

func (j *jwtService) ValidateToken(token string) (*jwt.Token, error) {
	t, err := jwt.ParseWithClaims(token, &jwtCustomClaim{}, j.parseToken)
	if err != nil {
		return nil, err
	}
	// exp tidak wajib di jwt v4, jadi diwajibkan di sini bersama issuer dan user_id.
	c, ok := t.Claims.(*jwtCustomClaim)
	if !ok || c.ExpiresAt == nil || c.Issuer != j.issuer || c.UserID == "" {
		return nil, fmt.Errorf("token tidak valid")
	}
	return t, nil
}

func (j *jwtService) GetUserIDByToken(token string) (string, error) {
	tToken, err := j.ValidateToken(token)
	if err != nil {
		return "", err
	}

	return tToken.Claims.(*jwtCustomClaim).UserID, nil
}
