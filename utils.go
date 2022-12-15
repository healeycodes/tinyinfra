package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

func newToken32() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(b)), nil
}

func APIError(route string, err error, w http.ResponseWriter, code int) {
	log.Printf("%v: error %v", route, err)
	w.WriteHeader(code)
}

type authError struct{}

func (e *authError) Error() string {
	return "Bad authentication attempt"
}

func auth(db *gorm.DB, r *http.Request) (*User, error) {
	prefix := "Bearer "
	authHeader := r.Header.Get("Authorization")
	reqToken := strings.TrimPrefix(authHeader, prefix)

	var user User
	result := db.Where("token = ?", reqToken).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, &authError{}
	}
	return &user, nil
}
