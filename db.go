package main

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Token string
}

type KVItem struct {
	gorm.Model
	Key    string
	Value  string
	TTL    int // UnixMilli, -1 is do not expire
	UserID int
	User   User
}

type QueueItem struct {
	gorm.Model
	Namespace string
	Message   string
	VisibleAt int // UnixMilli, item is visible if time > visible_at
	UserID    int
	User      User
}

type GetDBOptions struct {
	testing bool
	local   bool
	// TODO: production bool
}

func getDB(options GetDBOptions) *gorm.DB {
	var db *gorm.DB
	var err error

	if options.testing {
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	} else if options.local {
		db, err = gorm.Open(sqlite.Open("localdev.db"), &gorm.Config{})
	}
	// TODO: add "if production" branch

	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&User{}, &KVItem{}, &QueueItem{})
	return db
}
