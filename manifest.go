package arseeding

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/everFinance/arseeding/argraphql"
	"gopkg.in/h2non/gentleman.v2"
	"io"
	"net/http"
	"os"
	"strconv"
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

func getArTxOrItemDataForManifest(id string, db *Store, s *Arseeding) (decodeTags []types.Tag, binaryReader *os.File, data []byte, err error) {

	//  find bundle item form local
	decodeTags, binaryReader, data, err = getArTxOrItemData(id, db)

	//  if err equal ErrLocalNotExist, put task to queue to sync data
	if err == schema.ErrLocalNotExist {

		// registerTask
		if err = s.registerTask(id, schema.TaskTypeSyncManifest); err != nil {
			log.Error("registerTask  sync_manifest error", "err", err)
		}
		return nil, nil, nil, schema.ErrLocalNotExist
	}

	return decodeTags, binaryReader, data, err
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

func syncManifestData(id string, s *Arseeding) (err error) {

	//  get manifest  data
	data, contentType, err := getRawById(id)
	if err != nil {
		return err
	}

	// not manifest data
	if contentType != "application/x.arweave-manifest+json" {
		bundleItem, err := getBundleItem(id, data)
		if err != nil {
			return err
		}

		err = s.store.AtomicSaveItem(bundleItem)

		return err
	}

	//is manifest data parse it
	mani := schema.ManifestData{}
	if err := json.Unmarshal(data, &mani); err != nil {
		return err
	}

	log.Debug("total txId in manifest", "len", len(mani.Paths))
	// for each path in manifest
	for _, txId := range mani.Paths {
		log.Debug("syncManifestData", "txId", txId)
		// getBundleItem
		bundleItem, err := getBundleItem(txId.TxId, nil)

		if err != nil {
			return err
		}

		err = s.store.AtomicSaveItem(bundleItem)

	}

	return err
}

func getRawById(id string) (data []byte, contentType string, err error) {

	var arGateway = "https://arweave.net"
	client := gentleman.New().URL(arGateway)

	res, err := client.Get().AddPath(fmt.Sprintf("/raw/%s", id)).Send()

	if err != nil {
		return nil, "", err
	}

	if !res.Ok {
		return nil, "", fmt.Errorf("get raw data statuscode: %d  id: %s", res.StatusCode, id)
	}

	contentType = res.Header.Get("Content-Type")
	data = res.Bytes()

	return

}

func getBundleItem(id string, data []byte) (item types.BundleItem, err error) {
	// query  metadata from graphql
	gq := argraphql.NewARGraphQL("https://arweave-search.goldsky.com/graphql", http.Client{})

	txResp, err := gq.QueryTransaction(context.Background(), id)

	if err != nil {
		return item, err
	}
	tx := txResp.Transaction

	// cover tx.Tags to bundle item tags
	var tags []types.Tag
	for _, tag := range tx.Tags {
		tags = append(tags, types.Tag{
			Name:  tag.Name,
			Value: tag.Value,
		})
	}

	// cover size to int
	size, err := strconv.Atoi(tx.Data.Size)

	if err != nil {
		return item, err
	}

	// id data empty, get raw data
	if data == nil && size > 0 {
		// get raw data
		data, _, err = getRawById(id)
		if err != nil {
			return item, err
		}
	}

	// cover tags to tagsBy
	tagsBy, err := utils.SerializeTags(tags)

	if err != nil {
		return item, err
	}

	item = types.BundleItem{
		SignatureType: 1, // todo
		Signature:     tx.Signature,
		Owner:         tx.Owner.Key,
		Target:        "", //
		Anchor:        tx.Anchor,
		Tags:          tags,
		Data:          string(data), // raw api data
		Id:            tx.Id,
		TagsBy:        string(tagsBy), //
		ItemBinary:    nil,
		DataReader:    nil,
	}

	return item, nil

}
