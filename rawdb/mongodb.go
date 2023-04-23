package rawdb

import (
	"context"
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"reflect"
)

type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
	ctx      context.Context
}

type document struct {
	ID    string      `bson:"_id,omitempty"`
	Value interface{} `bson:"_value"`
}

const (
	K           = "_id"
	V           = "_value"
	MongoDBType = "MongoDB"
	dbName      = "arseeding"
)

// NewMongoDB uri be like mongodb://user:password@localhost:27017
func NewMongoDB(ctx context.Context, uri string) (*MongoDB, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}
	//test connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	log.Info("Connected to MongoDB!")
	return &MongoDB{client: client, database: client.Database(dbName), ctx: ctx}, nil
}

func (m *MongoDB) Put(bucket, key string, value interface{}) (err error) {
	if _, ok := value.([]byte); !ok {
		return fmt.Errorf("unknown data type: %s, db: MongoDB", reflect.TypeOf(value))
	}
	doc := document{
		ID:    key,
		Value: value,
	}
	if m.Exist(bucket, key) {
		filter := bson.D{{K, key}}
		update := bson.D{
			{"$set", bson.D{{V, value}}},
		}
		_, err = m.database.Collection(bucket).UpdateOne(m.ctx, filter, update)
		log.Info("Update Key >", K, key)
		return
	}
	objID, err := m.database.Collection(bucket).InsertOne(m.ctx, doc)
	log.Info("Put Key >", K, objID.InsertedID)
	return err
}

func (m *MongoDB) Get(bucket, key string) (data []byte, err error) {
	doc := document{}
	filter := bson.D{{K, key}}
	err = m.database.Collection(bucket).FindOne(m.ctx, filter).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return doc.Value.(primitive.Binary).Data, nil
}

func (m *MongoDB) GetAllKey(bucket string) (keys []string, err error) {
	cursor, err := m.database.Collection(bucket).Find(m.ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(m.ctx)
	var documents []document
	err = cursor.All(m.ctx, &documents)
	if err != nil {
		return nil, err
	}
	for _, d := range documents {
		keys = append(keys, d.ID)
	}
	return
}

func (m *MongoDB) Delete(bucket, key string) (err error) {
	filter := bson.D{{K, key}}
	_, err = m.database.Collection(bucket).DeleteMany(m.ctx, filter)
	return err
}

func (m *MongoDB) Close() (err error) {
	return m.client.Disconnect(m.ctx)
}

func (m *MongoDB) Type() string {
	return MongoDBType
}

func (m *MongoDB) Exist(bucket, key string) bool {
	filter := bson.D{{K, key}}
	err := m.database.Collection(bucket).FindOne(m.ctx, filter).Decode(&document{})
	//err == mongo.ErrNoDocuments {
	return err == nil
}

func (m *MongoDB) GetStream(bucket, key string) (data *os.File, err error) {
	return nil, schema.ErrNotImplement
}
