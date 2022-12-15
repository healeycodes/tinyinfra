package main

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"
)

type Key struct {
	Key string `json:"key"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func setKey(db *gorm.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := auth(db, r)
		if _, ok := err.(*authError); ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else if err != nil {
			APIError("setKey", err, w, http.StatusInternalServerError)
			return
		}

		var kv KeyValue
		err = json.NewDecoder(r.Body).Decode(&kv)
		if err != nil {
			APIError("setKey", err, w, http.StatusBadRequest)
			return
		}

		if err = db.Create(&KVItem{UserID: int(user.ID), Key: kv.Key, Value: kv.Value}).Error; err != nil {
			APIError("setKey", err, w, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func getKey(db *gorm.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := auth(db, r)
		if _, ok := err.(*authError); ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else if err != nil {
			APIError("getKey", err, w, http.StatusInternalServerError)
			return
		}

		var k Key
		err = json.NewDecoder(r.Body).Decode(&k)
		if err != nil {
			APIError("getKey", err, w, http.StatusBadRequest)
			return
		}

		var kvItem KVItem
		result := db.Where("user_id = ? AND key = ?", user.ID, k.Key).First(&kvItem)
		if result.Error != nil {
			APIError("getKey", result.Error, w, http.StatusInternalServerError)
			return
		} else if result.RowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&KeyValue{
			Key:   kvItem.Key,
			Value: kvItem.Value,
		})
	}
}
