package rawdb

import (
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestBoltDB(t *testing.T) {
	dataPath := "./tmp/seed.db"
	bktName := schema.ConstantsBucket // cne be replaced by any bucket in schema
	keyNum := 100
	// prepare key&val to test
	keys := make([]string, keyNum)
	values := make([][]byte, keyNum)
	for i := 0; i < keyNum; i++ {
		key := fmt.Sprintf("key%d", i)
		keys[i] = key
		val := fmt.Sprintf("v%d", i)
		values[i] = []byte(val)
	}
	assert.Equal(t, keyNum, len(keys))
	// create a bolt db
	boltDb, err := NewBoltDB(dataPath)
	assert.NoError(t, err)

	// test Put & Get
	for i := 0; i < keyNum; i++ {
		err = boltDb.Put(bktName, keys[i], values[i])
		assert.NoError(t, err)
	}

	for i := 0; i < keyNum; i++ {
		val, err := boltDb.Get(bktName, keys[i])
		assert.NoError(t, err)
		assert.Equal(t, values[i], val)
	}

	// test GetAllKey from a bucket
	allKeys, err := boltDb.GetAllKey(bktName)
	// GetAllKey return order may different from keys
	sort.Strings(allKeys)
	sort.Strings(keys)
	assert.NoError(t, err)
	assert.Equal(t, keys, allKeys)

	// test Delete
	for i := 0; i < keyNum; i++ {
		err = boltDb.Delete(bktName, keys[i])
		assert.NoError(t, err)
	}
	for i := 0; i < keyNum; i++ {
		_, err = boltDb.Get(bktName, keys[i])
		assert.Equal(t, err, schema.ErrNotExist)
	}
}

// func TestS3DB(t *testing.T) {
//
// 	bktName := schema.ConstantsBucket // cne be replaced by any bucket in schema
// 	keyNum := 10
// 	// prepare key&val to test
// 	keys := make([]string, keyNum)
// 	values := make([][]byte, keyNum)
// 	for i := 0; i < keyNum; i++ {
// 		key := fmt.Sprintf("key%d", i)
// 		keys[i] = key
// 		val := fmt.Sprintf("v%d", i)
// 		values[i] = []byte(val)
// 	}
// 	assert.Equal(t, keyNum, len(keys))
// 	// info that s3 needed
// 	accKey := "AKIATZSGGOHIV4QTYNH5"                        // your aws IAM access key
// 	secretKey := "uw3gKyHIZlaBx8vnCA/BSdNdH+Fi2j4ACoPJawOy" // your aws IAM secret key
// 	prefix := "arseed"                                      // create empty bucket
// 	Region := "ap-northeast-1"
// 	// create S3DB
// 	s, err := NewS3DB(accKey, secretKey, Region, prefix)
// 	// if the bucket exist try a complex prefix, because the bucket name is unique in a specific aws region
// 	assert.NoError(t, err)
//
// 	// test Put & Get
// 	for i := 0; i < keyNum; i++ {
// 		err = s.Put(bktName, keys[i], values[i])
// 		assert.NoError(t, err)
// 	}
//
// 	for i := 0; i < keyNum; i++ {
// 		val, err := s.Get(bktName, keys[i])
// 		assert.NoError(t, err)
// 		assert.Equal(t, values[i], val)
// 	}
//
// 	// test GetAllKey from a bucket
// 	allKeys, err := s.GetAllKey(bktName)
// 	assert.NoError(t, err)
// 	// GetAllKey return order may different from keys
// 	sort.Strings(allKeys)
// 	sort.Strings(keys)
// 	if len(allKeys) == len(keys) { // maybe s3 bucket not empty before test
// 		assert.Equal(t, keys, allKeys)
// 	}
//
// 	// test Delete
// 	for i := 0; i < keyNum; i++ {
// 		err = s.Delete(bktName, keys[i])
// 		assert.NoError(t, err)
// 	}
// 	for i := 0; i < keyNum; i++ {
// 		_, err = s.Get(bktName, keys[i])
// 		assert.Equal(t, err, schema.ErrNotExist)
// 	}
// }
