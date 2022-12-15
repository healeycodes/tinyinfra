package main

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"
)

func createUser(db *gorm.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := newToken32()
		if err != nil {
			APIError("createUser", err, w, http.StatusInternalServerError)
			return
		}
		newUser := &User{Token: token}
		if err := db.Create(newUser).Error; err != nil {
			APIError("createUser", err, w, http.StatusInternalServerError)
			return
		}

		type tokenResponse struct {
			Token string `json:"token"`
		}
		tokRes := tokenResponse{Token: token}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(tokRes)
	}
}
