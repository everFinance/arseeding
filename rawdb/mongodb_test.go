package rawdb

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"testing"
)

type Test struct {
	Name string
	Exp  string
}

func TestNewMongoDB(t *testing.T) {
	db, err := NewMongoDB(context.TODO(), "mongodb://root:password@localhost:27017", "arseeding")
	if err != nil {
		fmt.Println("Error")
		fmt.Println(err.Error())
		return
	}
	t1 := uuid.New().String()
	t2 := uuid.New().String()
	t3 := uuid.New().String()
	t4 := uuid.New().String()
	err = db.Put("arseeding", t2, []byte("string"))
	get1, err := db.Get("arseeding", t1)
	get4, err := db.Get("arseeding", t4)
	get2, err := db.Get("arseeding", t2)
	get3, err := db.Get("arseeding", t3)
	fmt.Println(string(get1))
	fmt.Println(string(get2))
	fmt.Println(string(get3))
	fmt.Println(string(get4))
	if err != nil {
		fmt.Println("Error")
		fmt.Println(err.Error())
		return
	}
}
