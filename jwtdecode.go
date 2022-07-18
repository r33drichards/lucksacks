package main

import (
	"fmt"

	"github.com/golang-jwt/jwt"
)

// https://stackoverflow.com/questions/45405626/decoding-jwt-token-in-golang
func jwtdecode(tokenString string) (string, error) {
	msg := ""
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, nil)
	// ... error handling
	if err != nil {
		return "", err
	}

	// do something with decoded claims
	for key, val := range claims {
		msg = msg + fmt.Sprintf("Key: %v, value: %v\n", key, val)
	}
	return msg, nil
}
