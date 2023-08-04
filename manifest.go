package arseeding

import (
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
)

func handleManifest(maniData []byte, path string, db *Store) ([]types.Tag, []byte, error) {
	mani := schema.ManifestData{}
	if err := json.Unmarshal(maniData, &mani); err != nil {
		return nil, nil, err
	}

	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		if mani.Index.Path == "" {
			return nil, nil, schema.ErrPageNotFound
		}
		path = mani.Index.Path
	}
	txId, ok := mani.Paths[path]
	if !ok {
		// could ignore index.html, so add index.html and try again
		txId, ok = mani.Paths[path+"/"+"index.html"]
	}
	if !ok {
		return nil, nil, schema.ErrPageNotFound
	}

	tags, dataReader, data, err := getArTxOrItemData(txId.TxId, db)
	if dataReader != nil {
		data, err = io.ReadAll(dataReader)
		dataReader.Close()
		os.Remove(dataReader.Name())
	}
	return tags, data, err
}

func getArTxOrItemData(id string, db *Store) (decodeTags []types.Tag, binaryReader *os.File, data []byte, err error) {
	// find bundle item
	_, err = db.LoadItemMeta(id)
	if err == nil {
		return getBundleItemData(id, db)
	}

	// not bundle item
	// find arId
	txMeta, err := db.LoadTxMeta(id)
	if err == nil { // arTx id
		data, err = txDataByMeta(txMeta, db)
		if err != nil {
			return
		}
		decodeTags, err = utils.TagsDecode(txMeta.Tags)
		return
	}
	// txId not found in local, need proxy to gateway
	return nil, nil, nil, schema.ErrLocalNotExist
}

func getArTxOrItemDataForManifest(id string, db *Store) (decodeTags []types.Tag, binaryReader *os.File, data []byte, err error) {

	//  find bundle item form local
	decodeTags, binaryReader, data, err = getArTxOrItemData(id, db)

	//  if err equal ErrLocalNotExist, put task to queue to sync data
	if err == schema.ErrLocalNotExist {

	}
	return nil, nil, nil, schema.ErrLocalNotExist
}

func getArTxOrItemTags(id string, db *Store) (decodeTags []types.Tag, err error) {
	itemMeta, err := db.LoadItemMeta(id)
	if err == nil {
		decodeTags = itemMeta.Tags
		return
	}
	// not bundle item
	// find arId
	txMeta, err := db.LoadTxMeta(id)
	if err == nil { // arTx id
		decodeTags, err = utils.TagsDecode(txMeta.Tags)
		return
	}
	return nil, schema.ErrLocalNotExist
}

func getBundleItemData(id string, db *Store) (decodeTags []types.Tag, dataReader *os.File, data []byte, err error) {
	binaryReader, itemBinary, err := db.LoadItemBinary(id)
	item := &types.BundleItem{}
	if err == nil {
		item, err = parseBundleItem(binaryReader, itemBinary)
		if err != nil {
			return nil, nil, nil, err
		}
		decodeTags = item.Tags
		if binaryReader != nil {
			binaryReader.Close()
			os.Remove(binaryReader.Name())
			dataReader = item.DataReader
		} else {
			data, err = utils.Base64Decode(item.Data)
		}
		return
	}
	return
}

func parseBundleItem(binaryReader *os.File, itemBinary []byte) (item *types.BundleItem, err error) {
	if binaryReader != nil {
		item, err = utils.DecodeBundleItemStream(binaryReader)
	} else {
		item, err = utils.DecodeBundleItem(itemBinary)
	}
	return
}
