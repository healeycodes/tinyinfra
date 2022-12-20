package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// KVCron clears up expired keys every hour.
// Note: these expired keys are already "invisible"
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
			APIServerError("setKey", err, w)
			return
		}

		kv := &KeyValue{TTL: -1}
		err = json.NewDecoder(r.Body).Decode(&kv)
		if err != nil {
			APIUserError(w, "error parsing JSON")
			return
		} else if kv.Key == "" {
			APIUserError(w, "key must not be empty or missing")
			return
		}

		// TODO: Use an upsert instead of a transaction plus two queries!
		err = db.Transaction(func(tx *gorm.DB) error {
			var ki KVItem
			if err = tx.Where("user_id = ? AND key = ?", user.ID, kv.Key).First(&ki).Error; err != nil {
				return tx.Create(&KVItem{UserID: int(user.ID), Key: kv.Key, Value: kv.Value, TTL: kv.TTL}).Error
			}
			return tx.Model(&ki).Updates(KVItem{Value: kv.Value, TTL: kv.TTL}).Error
		})
		if err != nil {
			APIServerError("setKey", err, w)
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
			APIServerError("getKey", err, w)
			return
		}

		var k Key
		err = json.NewDecoder(r.Body).Decode(&k)
		if err != nil {
			APIUserError(w, "error parsing JSON")
			return
		}
		if k.Key == "" {
			APIUserError(w, "expected key to be non-empty")
			return
		}

		var kvItem KVItem
		err = db.Where("user_id = ? AND key = ? AND (ttl = -1 OR ttl >= ?)", user.ID, k.Key, time.Now().UnixMilli()).First(&kvItem).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			APIServerError("getKey", err, w)
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
