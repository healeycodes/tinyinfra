package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestCreateUser(t *testing.T) {
	db := getDB(GetDBOptions{testing: true})

	req := httptest.NewRequest(http.MethodGet, "/user/new", nil)
	w := httptest.NewRecorder()
	createUser(db)(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("expected 200 got %v", res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	expression := `{"token":"[A-Za-z0-9+/]*={0,2}"}`

	// Check the returned token
	match, _ := regexp.MatchString(expression, string(data))
	if !match {
		t.Errorf("expected response to match %v got %v", expression, string(data))
	}

	// Check the user was created
	var users []User
	db.Find(&users)
	if len(users) != 1 {
		t.Errorf("expected to find one user got %v", len(users))
	}
	// With a non-empty token
	if users[0].Token == "" {
		t.Errorf("expected user token not to be empty")
	}
}
