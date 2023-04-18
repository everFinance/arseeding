package arseeding

import (
	"github.com/everFinance/goar/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestTempFile(t *testing.T) {
	file, err := os.CreateTemp(".", "arseed")
	assert.NoError(t, err)
	t.Log(file.Name())
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()
}

func TestDataSizeByStream(t *testing.T) {
	itembinary, err := ioutil.ReadFile("test.item")
	assert.NoError(t, err)
	item0, err := utils.DecodeBundleItem(itembinary)
	assert.NoError(t, err)
	assert.Equal(t, item0.DataReader, nil)
	binaryStream, err := os.Open("test.item")
	defer binaryStream.Close()
	assert.NoError(t, err)
	item1, err := utils.DecodeBundleItemStream(binaryStream)
	assert.NoError(t, err)
	dataStartCursor, err := item1.DataReader.Seek(0, 1)
	assert.NoError(t, err)
	fileInfo, err := item1.DataReader.Stat()
	assert.NoError(t, err)
	dataBy, err := utils.Base64Decode(item0.Data)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(dataBy)), fileInfo.Size()-dataStartCursor)
}
