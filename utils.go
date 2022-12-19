package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
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

func APIServerError(route string, err error, w http.ResponseWriter) {
	log.Printf("%v: error %v", route, err)
	w.WriteHeader(http.StatusInternalServerError)
}

type UserError struct {
	Message string `json:"message"`
}

func APIUserError(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&UserError{
		Message: message,
	})
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
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, &authError{}
	} else if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}
