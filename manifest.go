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

	log.Error("getArTxOrItemDataForManifest getArTxOrItemData", "err", err)
	//  if err equal ErrLocalNotExist, put task to queue to sync data
	if err == schema.ErrLocalNotExist {
		// registerTask
		if err = s.registerTask(id, schema.TaskTypeSyncManifest); err != nil {
			log.Error("registerTask  sync_manifest error", "err", err)
		}
		log.Debug("registerTask sync_manifest...", "id", id)
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

	allItemIds := []string{id}
	// if contentType == "application/x.arweave-manifest+json"  parse it
	if contentType == "application/x.arweave-manifest+json" {
		// is manifest data parse it
		mani := schema.ManifestData{}
		if err := json.Unmarshal(data, &mani); err != nil {
			return err
		}

		log.Debug("total txId in manifest", "len", len(mani.Paths))
		// for each path in manifest
		for _, txId := range mani.Paths {
			allItemIds = append(allItemIds, txId.TxId)
		}
	}

	log.Debug("allItemIds", "ids", len(allItemIds))

	// filter exist item
	itemIds := make([]string, 0, 10)
	for _, i := range allItemIds {
		if s.store.IsExistItemBinary(i) {
			log.Debug("syncManifestData existItem", "item", i)
			continue
		}
		itemIds = append(itemIds, i)
	}

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
		log.Debug("BatchGetItemsBundleIn", "start", i, "end", end, "itemIds[i:end]", len(itemIds[i:end]))
		resp, err := gq.BatchGetItemsBundleIn(context.Background(), itemIds[i:end], 90, "")
		if err != nil {
			return errors.New("BatchGetItemsBundleIn error:" + err.Error())
		}
		txs = append(txs, resp.Transactions.Edges...)
	}
	log.Debug("txs", "txs", len(txs))

	//  for each txs to   bundleInItemsMap
	var (
		bundleInItemsMap = make(map[string][]string)
		L1Artxs          = make([]string, 0)
	)
	for _, tx := range txs {
		if tx.Node.BundledIn.Id == "" {
			L1Artxs = append(L1Artxs, tx.Node.Id)
		} else {
			if _, ok := bundleInItemsMap[tx.Node.BundledIn.Id]; !ok {
				bundleInItemsMap[tx.Node.BundledIn.Id] = make([]string, 0)
			}
			bundleInItemsMap[tx.Node.BundledIn.Id] = append(bundleInItemsMap[tx.Node.BundledIn.Id], tx.Node.Id)
		}
	}
	log.Debug("syncManifestData bundleInItemsMap", "bundleInItemsMap", len(bundleInItemsMap), "L1Artxs", len(L1Artxs))
	// get bundle item  form goar
	c := goar.NewClient("https://arweave.net")
	for bundleId, itemIdss := range bundleInItemsMap {
		log.Debug("syncManifestData GetBundleItems", "bundleId", bundleId, "itemIds", len(itemIdss))
		var items []*types.BundleItem
		// check bundleId is nestBundle
		isNestBundle, dataSize, err := checkNestBundle(bundleId, gq)
		if err != nil || !isNestBundle {
			// GetBundleItems
			items, err = c.GetBundleItems(bundleId, itemIdss)
			if err != nil {
				return errors.New("GetBundleItems error:" + err.Error())
			}
		} else {
			log.Debug("nestBundle dataSize...", "size", dataSize)
			items, err = getNestBundle(bundleId, itemIdss)
			if err != nil {
				return errors.New("getNestBundle error:" + err.Error())
			}
		}

		//  for each item
		for _, item := range items {
			log.Debug("syncManifestData save ", "item id", item.Id)
			// save item to store
			if err := s.saveItem(*item); err != nil {
				return fmt.Errorf("saveItem failed, err: %v", err)
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

func getNestBundle(nestBundle string, itemIds []string) (items []*types.BundleItem, err error) {
	data, _, err := getRawById(nestBundle)
	if err != nil {
		return nil, err
	}
	bundle, err := utils.DecodeBundle(data)
	if err != nil {
		return nil, err
	}
	for _, item := range bundle.Items {
		if utils.ContainsInSlice(itemIds, item.Id) {
			itemBy, err := utils.DecodeBundleItem(item.ItemBinary)
			if err != nil {
				return nil, err
			}
			items = append(items, itemBy)
		}
	}
	return
}

func checkNestBundle(bundleId string, gq *argraphql.ARGraphQL) (isNestBundle bool, dataSize string, err error) {
	res, err := gq.QueryTransaction(context.Background(), bundleId)
	if err != nil {
		return
	}
	isNestBundle = res.Transaction.BundledIn.Id != ""
	dataSize = res.Transaction.Data.Size
	return
}
