package rawdb

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"testing"
)

func TestNewMongoDB(t *testing.T) {
	db, err := NewMongoDB(context.TODO(), "mongodb://root:pwd@localhost:27017")
	if err != nil {
		fmt.Println("Error TestNewMongoDB")
		fmt.Println(err.Error())
		return
	}
	t1 := uuid.New().String()
	err = db.Put("arseeding", "testKey", []byte("test msg"))
	err = db.Put("arseeding", t1, []byte("old"))
	get1, err := db.Get("arseeding", t1)
	fmt.Println(string(get1))

	err = db.Put("arseeding", t1, []byte("new"))
	get1, err = db.Get("arseeding", t1)
	fmt.Println(string(get1))

	if err != nil {
		fmt.Println("Error")
		fmt.Println(err.Error())
		return
	}
}

func TestGet(t *testing.T) {
	db, _ := NewMongoDB(context.TODO(), "mongodb://root:pwd@localhost:27017")
	allKey, _ := db.GetAllKey("arseeding")
	fmt.Println(len(allKey))
	fmt.Println(allKey)
}

func TestDel(t *testing.T) {
	db, _ := NewMongoDB(context.TODO(), "mongodb://root:pwd@localhost:27017")
	err := db.Delete("arseeding", "testKey")
	if err != nil {
		fmt.Println(err.Error())
	}
}
