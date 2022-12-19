package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetKey(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	token := "a"
	db.Create(&User{Token: token})

	req := httptest.NewRequest(http.MethodGet, "/kv/set", ioutil.NopCloser(strings.NewReader(`{"key": "some_key", "value": "some_value", "ttl": 1986589728969}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	setKey(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("expected 200 got %v", res.StatusCode)
	}

	// Check the KVItem was created
	var kvItems []KVItem
	db.Preload("User").Find(&kvItems)
	if len(kvItems) != 1 {
		t.Errorf("expected to find one item got %v", len(kvItems))
	}
	if kvItems[0].Key != "some_key" || kvItems[0].Value != "some_value" || kvItems[0].TTL != 1986589728969 || kvItems[0].User.ID != 1 {
		t.Errorf("expected item to be created correctly got %v %v %v %v", kvItems[0].Key, kvItems[0].Value, kvItems[0].TTL, kvItems[0].User.ID)
	}
}

func TestSetKeyUpdate(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	token := "a"
	user := &User{Token: token}
	db.Create(&user)
	db.Create(&KVItem{Key: "some_key", Value: "some_value", TTL: -1, UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/kv/set", ioutil.NopCloser(strings.NewReader(`{"key": "some_key", "value": "some_value2", "ttl": 1986589728969}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	setKey(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("expected 200 got %v", res.StatusCode)
	}

	// Check the KVItem was updated
	var kvItems []KVItem
	db.Preload("User").Find(&kvItems)
	if len(kvItems) != 1 {
		t.Errorf("expected to find one item got %v", len(kvItems))
	}
	if kvItems[0].Key != "some_key" || kvItems[0].Value != "some_value2" || kvItems[0].TTL != 1986589728969 || kvItems[0].User.ID != 1 {
		t.Errorf("expected item to be created correctly got %v %v %v %v", kvItems[0].Key, kvItems[0].Value, kvItems[0].TTL, kvItems[0].User.ID)
	}
}

func TestSetKeyBadAuth(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	token := "a"
	db.Create(&User{Token: token})

	req := httptest.NewRequest(http.MethodGet, "/kv/set", ioutil.NopCloser(strings.NewReader(`{"key": "some_key", "value": "some_value", "ttl": 1986589728969}`)))
	req.Header.Set("Authorization", "Bearer b")
	w := httptest.NewRecorder()
	setKey(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 401 {
		t.Errorf("expected 401 got %v", res.StatusCode)
	}
}

func TestGetKeyBadAuth(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)
	db.Create(&KVItem{Key: "some_key", Value: "some_value", TTL: 1986589728969, UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/kv/get", ioutil.NopCloser(strings.NewReader(`{"key": "some_key"}`)))
	req.Header.Set("Authorization", "Bearer b")
	w := httptest.NewRecorder()
	getKey(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 401 {
		t.Errorf("expected 401 got %v", res.StatusCode)
	}
}

func TestGetKeyNoExpire(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)
	db.Create(&KVItem{Key: "some_key", Value: "some_value", TTL: -1, UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/kv/get", ioutil.NopCloser(strings.NewReader(`{"key": "some_key"}`)))
	req.Header.Set("Authorization", "Bearer "+user.Token)
	w := httptest.NewRecorder()
	getKey(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("expected 200 got %v", res.StatusCode)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	var kv KeyValue
	err = json.Unmarshal(data, &kv)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	// Check the key/value is correct
	if kv.Key != "some_key" || kv.Value != "some_value" || kv.TTL != -1 {
		t.Errorf("expected to find correct key/value got %v %v %v", kv.Key, kv.Value, kv.TTL)
	}
}

func TestGetKeyExpiresInFuture(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)
	db.Create(&KVItem{Key: "some_key", Value: "some_value", TTL: 1986589728969, UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/kv/get", ioutil.NopCloser(strings.NewReader(`{"key": "some_key"}`)))
	req.Header.Set("Authorization", "Bearer "+user.Token)
	w := httptest.NewRecorder()
	getKey(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("expected 200 got %v", res.StatusCode)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	var kv KeyValue
	err = json.Unmarshal(data, &kv)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	// Check the key/value is correct
	if kv.Key != "some_key" || kv.Value != "some_value" || kv.TTL != 1986589728969 {
		t.Errorf("expected to find correct key/value got %v %v %v", kv.Key, kv.Value, kv.TTL)
	}
}

func TestGetKeyExpired(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)
	db.Create(&KVItem{Key: "some_key", Value: "some_value", TTL: 1, UserID: int(user.ID)})

	req := httptest.NewRequest(http.MethodGet, "/kv/get", ioutil.NopCloser(strings.NewReader(`{"key": "some_key"}`)))
	req.Header.Set("Authorization", "Bearer "+user.Token)
	w := httptest.NewRecorder()
	getKey(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 404 {
		t.Errorf("expected 404 got %v", res.StatusCode)
	}
}

func TestGetKeyMissing(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})
	user := &User{Token: "a"}
	db.Create(user)

	req := httptest.NewRequest(http.MethodGet, "/kv/get", ioutil.NopCloser(strings.NewReader(`{"key": "some_key"}`)))
	req.Header.Set("Authorization", "Bearer "+user.Token)
	w := httptest.NewRecorder()
	getKey(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 404 {
		t.Errorf("expected 404 got %v", res.StatusCode)
	}
}
