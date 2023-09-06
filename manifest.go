package arseeding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/everFinance/arseeding/argraphql"
	"github.com/everFinance/goar"
	"gopkg.in/h2non/gentleman.v2"
	"io"
	"net/http"
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

	log.Debug("syncManifestData get raw end ", "id", id, "contentType", contentType, "data", string(data))
	var bundleInItemsMap = make(map[string][]string)
	var L1Artxs []string
	var itemIds []string

	itemIds = append(itemIds, id)

	// if contentType == "application/x.arweave-manifest+json"  parse it
	if contentType == "application/x.arweave-manifest+json" {

		//is manifest data parse it
		mani := schema.ManifestData{}
		if err := json.Unmarshal(data, &mani); err != nil {
			return err
		}

		log.Debug("total txId in manifest", "len", len(mani.Paths))
		// for each path in manifest
		for _, txId := range mani.Paths {
			itemIds = append(itemIds, txId.TxId)
		}
	}

	log.Debug("total txId in manifest", "len", len(itemIds))

	// query itemIds from graphql
	gq := argraphql.NewARGraphQL("https://arweave.net/graphql", http.Client{})

	total := len(itemIds)

	var txs []argraphql.BatchGetItemsBundleInTransactionsTransactionConnectionEdgesTransactionEdge
	// 90 items per query
	for i := 0; i < total; i += 90 {
		end := i + 90
		if end > total {
			end = total
		}
		log.Debug("BatchGetItemsBundleIn", "start", i, "end", end)
		resp, err := gq.BatchGetItemsBundleIn(context.Background(), itemIds[i:end], 90, "")
		if err != nil {
			return errors.New("BatchGetItemsBundleIn error:" + err.Error())
		}
		txs = append(txs, resp.Transactions.Edges...)
	}

	//  for each txs to   bundleInItemsMap
	for _, tx := range txs {
		if tx.Node.BundledIn.Id == "" {
			L1Artxs = append(L1Artxs, tx.Node.Id)
		} else {
			bundleInItemsMap[tx.Node.BundledIn.Id] = append(bundleInItemsMap[tx.Node.BundledIn.Id], tx.Node.Id)
		}
	}

	log.Debug("syncManifestData bundleInItemsMap", "bundleInItemsMap", bundleInItemsMap, "L1Artxs", L1Artxs)
	// get bundle item  form goar
	c := goar.NewClient("https://arweave.net")
	for bundleId, itemIds := range bundleInItemsMap {

		log.Debug("syncManifestData GetBundleItems ", "bundleId", bundleId, "itemIds", itemIds)
		// GetBundleItems
		items, err := c.GetBundleItems(bundleId, itemIds)

		if err != nil {
			return errors.New("GetBundleItems error:" + err.Error())
		}

		//  for each item
		for _, item := range items {
			log.Debug("syncManifestData save ", "item id", item.Id)
			// save item to store
			err = s.saveItem(*item)

			if err != nil {
				return errors.New("saveItem error:" + err.Error())
			}
		}

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
