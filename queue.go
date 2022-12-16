package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type QueueMessage struct {
	Namespace string `json:"namespace"`
	Message   string `json:"message"`
}

type QueueRequest struct {
	Namespace         string `json:"namespace"`
	VisibilityTimeout int    `json:"visibilityTimeout"`
}

type QueueResponse struct {
	ID        uint   `json:"id"`
	Namespace string `json:"namespace"`
	Message   string `json:"message"`
}

type QueueMessageToDelete struct {
	ID        string `json:"id"`
	Namespace string `json:"namespace"`
}

func sendMessage(db *gorm.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := auth(db, r)
		if _, ok := err.(*authError); ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else if err != nil {
			APIError("sendMessage", err, w, http.StatusInternalServerError)
			return
		}

		qm := &QueueMessage{}
		err = json.NewDecoder(r.Body).Decode(&qm)
		if err != nil {
			APIError("sendMessage", err, w, http.StatusBadRequest)
			return
		}
		// TODO: check and fail for bad data

		if err = db.Create(&QueueItem{UserID: int(user.ID), Namespace: qm.Namespace, Message: qm.Message, VisibleAt: 0}).Error; err != nil {
			APIError("sendMessage", err, w, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func receiveMessage(db *gorm.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := auth(db, r)
		if _, ok := err.(*authError); ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else if err != nil {
			APIError("receiveMessage", err, w, http.StatusInternalServerError)
			return
		}

		var qr QueueRequest
		err = json.NewDecoder(r.Body).Decode(&qr)
		if err != nil {
			APIError("receiveMessage", err, w, http.StatusBadRequest)
			return
		}
		// TODO: handle bad data

		var queueItem QueueItem
		err = db.Transaction(func(tx *gorm.DB) error {
			if err = tx.Where("user_id = ? AND namespace = ? AND visible_at >= ?",
				user.ID, qr.Namespace, time.Now().UnixMilli()).First(&queueItem).Error; err != nil {
				return err
			}
			queueItem.VisibleAt = int(time.Now().UnixMilli() + int64(qr.VisibilityTimeout))
			if err = tx.Save(queueItem).Error; err != nil {
				return err
			}
			return nil
		})

		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			APIError("getKey", err, w, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&QueueResponse{
			ID:        queueItem.ID,
			Namespace: queueItem.Namespace,
			Message:   queueItem.Message,
		})
	}
}

func deleteMessage(db *gorm.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := auth(db, r)
		if _, ok := err.(*authError); ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else if err != nil {
			APIError("receiveMessage", err, w, http.StatusInternalServerError)
			return
		}

		var qd QueueMessageToDelete
		err = json.NewDecoder(r.Body).Decode(&qd)
		if err != nil {
			APIError("receiveMessage", err, w, http.StatusBadRequest)
			return
		}
		// TODO: handle bad data

		err = db.Where("user_id = ? AND namespace = ? AND id = ?", user.ID, qd.Namespace, qd.ID).Delete(&QueueItem{}).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			APIError("getKey", err, w, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
