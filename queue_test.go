package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestSendMessage(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	token := "a"
	db.Create(&User{Token: token})

	req := httptest.NewRequest(http.MethodGet, "/queue/send", ioutil.NopCloser(strings.NewReader(`{"namespace": "a", "message": "b"}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	sendMessage(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("expected 200 got %v", res.StatusCode)
	}

	// Check the QueueItem was created
	var queueItems []QueueItem
	db.Preload("User").Find(&queueItems)
	if len(queueItems) != 1 {
		t.Errorf("expected to find one item got %v", len(queueItems))
	}
	if queueItems[0].Namespace != "a" || queueItems[0].Message != "b" || queueItems[0].UserID != 1 {
		t.Errorf("expected item to be created correctly got %v %v %v", queueItems[0].Namespace, queueItems[0].Message, queueItems[0].UserID)
	}
}

func TestSendMessageBadAuth(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	token := "a"
	db.Create(&User{Token: token})

	req := httptest.NewRequest(http.MethodGet, "/queue/send", ioutil.NopCloser(strings.NewReader(`{"namespace": "a", "message": "b"}`)))
	req.Header.Set("Authorization", "Bearer b")
	w := httptest.NewRecorder()
	sendMessage(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 401 {
		t.Errorf("expected 401 got %v", res.StatusCode)
	}
}

func TestReceiveMessage(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)
	db.Create(&QueueItem{Namespace: "a", Message: "b", VisibleAt: 0, UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/queue/receive", ioutil.NopCloser(strings.NewReader(`{"namespace": "a", "visibilityTimeout": 20000}`)))
	req.Header.Set("Authorization", "Bearer "+user.Token)
	w := httptest.NewRecorder()
	receiveMessage(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("expected 200 got %v", res.StatusCode)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	var qr QueueResponse
	err = json.Unmarshal(data, &qr)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	// Check the message is correct
	if qr.ID != 1 || qr.Namespace != "a" || qr.Message != "b" {
		t.Errorf("expected to find correct key/value got %v %v %v", qr.ID, qr.Namespace, qr.Message)
	}

	// Check the visibility timeout was correctly applied
	var qiItems []QueueItem
	db.Preload("User").Find(&qiItems)
	if len(qiItems) != 1 {
		t.Errorf("expected to find one item got %v", len(qiItems))
	}
	// Allow two seconds of leeway (the time it takes for the API call to happen)
	if qiItems[0].VisibleAt < int(time.Now().UnixMilli()+(18*1000)) {
		t.Errorf("expected visibility timeout to be applied to message wanted wanted > %v got %v (diff: %v)",
			time.Now().UnixMilli()+18000, qiItems[0].VisibleAt, int(time.Now().UnixMilli()+18000)-qiItems[0].VisibleAt)
	}
}

func TestReceiveMessageBadAuth(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)
	db.Create(&QueueItem{Namespace: "a", Message: "b", VisibleAt: 0, UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/queue/receive", ioutil.NopCloser(strings.NewReader(`{"namespace": "a", "visibilityTimeout": 20000}`)))
	req.Header.Set("Authorization", "Bearer b")
	w := httptest.NewRecorder()
	receiveMessage(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 401 {
		t.Errorf("expected 401 got %v", res.StatusCode)
	}
}

func TestReceiveEarlierMessage(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)
	db.Create(&QueueItem{Namespace: "a", Message: "b", VisibleAt: 0, UserID: int(user.ID)})
	db.Create(&QueueItem{Namespace: "a", Message: "c", VisibleAt: 0, UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/queue/receive", ioutil.NopCloser(strings.NewReader(`{"namespace": "a", "visibilityTimeout": 20000}`)))
	req.Header.Set("Authorization", "Bearer "+user.Token)
	w := httptest.NewRecorder()
	receiveMessage(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("expected 200 got %v", res.StatusCode)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	var qr QueueResponse
	err = json.Unmarshal(data, &qr)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	// Check we get the earlier message
	if qr.ID != 1 || qr.Namespace != "a" || qr.Message != "b" {
		t.Errorf("expected to find correct key/value got %v %v %v", qr.ID, qr.Namespace, qr.Message)
	}

	// Check the visibility timeout was correctly applied
	var qiItem QueueItem
	if err = db.Where("id = 1").First(&qiItem).Error; err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	// Allow two seconds of leeway (the time it takes for the API call to happen)
	if qiItem.VisibleAt < int(time.Now().UnixMilli()+(18*1000)) {
		t.Errorf("expected visibility timeout to be applied to message wanted wanted > %v got %v (diff: %v)",
			time.Now().UnixMilli()+18000, qiItem.VisibleAt, int(time.Now().UnixMilli()+18000)-qiItem.VisibleAt)
	}
}

func TestReceiveInvisibleMessage(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)
	db.Create(&QueueItem{Namespace: "a", Message: "b", VisibleAt: int(time.Now().UnixMilli() + (18 * 1000)), UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/queue/receive", ioutil.NopCloser(strings.NewReader(`{"namespace": "a", "visibilityTimeout": 20000}`)))
	req.Header.Set("Authorization", "Bearer "+user.Token)
	w := httptest.NewRecorder()
	receiveMessage(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 404 {
		t.Errorf("expected 404 got %v", res.StatusCode)
	}
}

func TestDeleteMessage(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)
	db.Create(&QueueItem{Namespace: "a", Message: "b", VisibleAt: int(time.Now().UnixMilli() + (18 * 1000)), UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/queue/delete", ioutil.NopCloser(strings.NewReader(`{"namespace": "a", "id": 1}`)))
	req.Header.Set("Authorization", "Bearer "+user.Token)
	w := httptest.NewRecorder()
	deleteMessage(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("expected 200 got %v", res.StatusCode)
	}

	// Check the item was deleted
	var qiItem QueueItem
	if err := db.Where("id = 1").First(&qiItem).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected item to be deleted")
	}
}

func TestDeleteMessageBadAuth(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)
	db.Create(&QueueItem{Namespace: "a", Message: "b", VisibleAt: int(time.Now().UnixMilli() + (18 * 1000)), UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/queue/delete", ioutil.NopCloser(strings.NewReader(`{"namespace": "a", "id": 1}`)))
	req.Header.Set("Authorization", "Bearer b")
	w := httptest.NewRecorder()
	deleteMessage(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 401 {
		t.Errorf("expected 401 got %v", res.StatusCode)
	}
}

func TestDeleteMissingMessage(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)
	db.Create(&QueueItem{Namespace: "a", Message: "b", VisibleAt: int(time.Now().UnixMilli() + (18 * 1000)), UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/queue/delete", ioutil.NopCloser(strings.NewReader(`{"namespace": "a", "id": 2}`)))
	req.Header.Set("Authorization", "Bearer "+user.Token)
	w := httptest.NewRecorder()
	deleteMessage(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 404 {
		t.Errorf("expected 404 got %v", res.StatusCode)
	}
}
