package rawdb

import (
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
)

type MongoDB struct {
}

func NewMongoDB() *MongoDB {
	options.Client()
	return nil
}

func (m *MongoDB) Put(bucket, key string, value interface{}) (err error) {
	//TODO implement me
	panic("implement me")
}

func (m *MongoDB) Get(bucket, key string) (data []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *MongoDB) GetStream(bucket, key string) (data *os.File, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *MongoDB) GetAllKey(bucket string) (keys []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *MongoDB) Delete(bucket, key string) (err error) {
	//TODO implement me
	panic("implement me")
}

func (m *MongoDB) Close() (err error) {
	//TODO implement me
	panic("implement me")
}

func (m *MongoDB) Type() string {
	//TODO implement me
	panic("implement me")
}

func (m *MongoDB) Exist(bucket, key string) bool {
	//TODO implement me
	panic("implement me")
}
