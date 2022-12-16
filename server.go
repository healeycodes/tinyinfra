package main

import (
	"net/http"
)

func main() {
	db := getDB(GetDBOptions{local: true})

	http.HandleFunc("/user/new", createUser(db))
	http.HandleFunc("/kv/set", setKey(db))
	http.HandleFunc("/kv/get", getKey(db))
	http.HandleFunc("/queue/send", sendMessage(db))

	KVCron(db)
	http.ListenAndServe(":8000", nil)
}
