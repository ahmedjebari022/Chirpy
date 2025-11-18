package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)


func HashPassword(password string) (string,error){
	hash, err := argon2id.CreateHash(password,argon2id.DefaultParams)
	if err != nil {
		return "",err
	}
	return hash,nil
}

func CheckPasswordHash(password, hash string) (bool, error){
	match, err := argon2id.ComparePasswordAndHash(password,hash)
	if err != nil {
		return match,err
	}
	return match,nil
}


func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration)(string, error){
	signingMethod := jwt.SigningMethodHS256
	claim := jwt.RegisteredClaims{
		Issuer: "chirpy",
		IssuedAt: jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject: userID.String(),
	}
	token := jwt.NewWithClaims(signingMethod,claim)
	jwt, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "",err
	}
	return jwt,nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error){
	token, err := jwt.ParseWithClaims(tokenString,&jwt.RegisteredClaims{}, func(token *jwt.Token)(any,error){
		// if _, ok := token.Method.(*jwt.SigningMethodHMAC);!ok{
		// 	return nil,fmt.Errorf("unexpected signing method")
		// }	
		return []byte(tokenSecret),nil
	})
	if err != nil {
		return uuid.UUID{},err
	}
	if !token.Valid{
		return uuid.Nil, fmt.Errorf("invalid token")
	}
	claims := token.Claims
	subject, err := claims.GetSubject()
	if err != nil {
		return uuid.UUID{},err
	}
	userId, err := uuid.Parse(subject)
	if err != nil {
		return uuid.UUID{},err
	}
	return userId,nil
}

func GetBearerToken(headers http.Header) (string, error){
	TOKEN_STRING := headers.Get("Authorization")
	if TOKEN_STRING == ""{
		return TOKEN_STRING,fmt.Errorf("no token provider in the headers")
	}
	TOKEN_STRING = strings.TrimPrefix(TOKEN_STRING,"Bearer ")
	return TOKEN_STRING,nil
}