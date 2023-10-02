package jwt

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func CreateToken(expires_in_seconds int, userId int, issuer string) (string, error) {
	if expires_in_seconds == 0 {
		expires_in_seconds = 3600
	}
	currentTime := time.Now()
	convertedExpiration := time.Second * time.Duration(expires_in_seconds)

	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   strconv.Itoa(userId),
		Audience:  nil,
		ExpiresAt: jwt.NewNumericDate(currentTime.Add(convertedExpiration)),
		NotBefore: nil,
		IssuedAt:  jwt.NewNumericDate(currentTime),
		ID:        "",
	})

	signedToken, err := unsignedToken.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func ValidateToken(tokenString string) (*jwt.Token, error) {
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return nil, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return nil, err
	}

	if issuer != "chirpy-refresh" {
		return nil, errors.New("invalid issuer")
	}

	return token, nil
}

func GetUserIdFromToken(tokenString *jwt.Token) (int, error) {
	idString, err := tokenString.Claims.GetSubject()
	if err != nil {
		return 0, err
	}

	id, err := strconv.Atoi(idString)
	if err != nil {
		return 0, err
	}

	return id, nil
}
