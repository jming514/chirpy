package jwt

import (
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func CreateToken(expires_in_seconds int, userId int) (string, error) {
    if expires_in_seconds == 0 {
        expires_in_seconds = 3600
    }
	currentTime := time.Now()
	convertedExpiration := time.Second * time.Duration(expires_in_seconds)

	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
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
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return nil, err
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
