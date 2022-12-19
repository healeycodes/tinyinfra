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
	ID        uint   `json:"id"`
	Namespace string `json:"namespace"`
}

func sendMessage(db *gorm.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := auth(db, r)
		if _, ok := err.(*authError); ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else if err != nil {
			APIServerError("sendMessage", err, w)
			return
		}

		qm := &QueueMessage{}
		err = json.NewDecoder(r.Body).Decode(&qm)
		if err != nil {
			APIUserError(w, "error parsing JSON")
			return
		}
		if qm.Namespace == "" || qm.Message == "" {
			APIUserError(w, "expected namespace and message to be non-empty")
			return
		}

		if err = db.Create(&QueueItem{UserID: int(user.ID), Namespace: qm.Namespace, Message: qm.Message, VisibleAt: 0}).Error; err != nil {
			APIServerError("sendMessage", err, w)
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
			APIServerError("receiveMessage", err, w)
			return
		}

		var qr QueueRequest
		err = json.NewDecoder(r.Body).Decode(&qr)
		if err != nil {
			APIUserError(w, "error parsing JSON")
			return
		}
		if qr.Namespace == "" {
			APIUserError(w, "expected namespace to be non-empty")
			return
		}

		var queueItem QueueItem
		err = db.Transaction(func(tx *gorm.DB) error {
			if err = tx.Where("user_id = ? AND namespace = ? AND (visible_at = 0 OR visible_at <= ?)",
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
			APIServerError("receiveMessage", err, w)
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
			APIServerError("deleteMessage", err, w)
			return
		}

		qd := &QueueMessageToDelete{ID: 0}
		err = json.NewDecoder(r.Body).Decode(&qd)
		if err != nil {
			APIUserError(w, "error parsing JSON")
			return
		}
		if qd.Namespace == "" || qd.ID == 0 {
			APIUserError(w, "expected namespace and id to be non-empty")
			return
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			var qi QueueItem
			if err = tx.Where("user_id = ? AND namespace = ? AND id = ?", user.ID, qd.Namespace, qd.ID).First(&qi).Error; err != nil {
				return err
			}
			err = tx.Delete(&qi).Error
			if err != nil {
				return err
			}
			return nil
		})
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			APIServerError("deleteMessage", err, w)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
