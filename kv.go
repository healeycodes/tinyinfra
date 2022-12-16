package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"
)

func KVCron(db *gorm.DB) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			<-ticker.C
			db.Delete(&KVItem{}, "TTL != -1 AND TTL < ?", time.Now().UnixMilli())
		}
	}()
}

type Key struct {
	Key string `json:"key"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
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

		kv := &KeyValue{TTL: -1}
		err = json.NewDecoder(r.Body).Decode(&kv)
		if err != nil {
			APIError("setKey", err, w, http.StatusBadRequest)
			return
		}
		// TODO: check and fail for bad data

		if err = db.Create(&KVItem{UserID: int(user.ID), Key: kv.Key, Value: kv.Value, TTL: kv.TTL}).Error; err != nil {
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
		err = db.Where("user_id = ? AND key = ? AND (ttl = -1 OR ttl >= ?) ", user.ID, k.Key, time.Now().UnixMilli()).First(&kvItem).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			APIError("getKey", err, w, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&KeyValue{
			Key:   kvItem.Key,
			Value: kvItem.Value,
			TTL:   kvItem.TTL,
		})
	}
}
