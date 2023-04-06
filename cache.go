package arseeding

import (
	"encoding/json"
	"errors"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"sort"
	"sync"
	"time"
)

type Cache struct {
	arInfo  types.NetworkInfo
	anchor  string
	fee     schema.ArFee
	peerMap map[string]int64   // available peer list ,those peers response quickly, key->value:peerIp->availableCount
	constTx *types.Transaction // a legal arTx used to test a peer is available
	lock    sync.RWMutex
}

func NewCache(arCli *goar.Client, peerMap map[string]int64) *Cache {
	c := &Cache{peerMap: peerMap}
	peers := c.GetPeers()
	arInfo, err := fetchArInfo(arCli, peers)
	if err != nil {
		panic(err)
	}
	c.UpdateInfo(*arInfo)

	fee, err := fetchArFee(arCli, peers)
	if err != nil {
		panic(err)
	}
	c.UpdateFee(fee)

	anchor, err := fetchAnchor(arCli, peers)
	if err != nil {
		panic(err)
	}
	c.UpdateAnchor(anchor)

	constTx := &types.Transaction{}
	if err := json.Unmarshal([]byte(schema.ConstTx), constTx); err != nil {
		panic(err)
	}
	c.constTx = constTx
	return c
}

func (c *Cache) GetInfo() types.NetworkInfo {
	c.lock.RLock()
	defer c.lock.RUnlock()
	info := c.arInfo
	return info
}

func (c *Cache) UpdateInfo(info types.NetworkInfo) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.arInfo = info
}

func (c *Cache) GetAnchor() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	anchor := c.anchor
	return anchor
}

func (c *Cache) UpdateAnchor(anchor string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.anchor = anchor
}

func (c *Cache) GetFee() schema.ArFee {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.fee
}

func (c *Cache) UpdateFee(price schema.ArFee) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.fee = price
}

func (c *Cache) GetPeerMap() map[string]int64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	peerMap := make(map[string]int64)
	for peerIp, count := range c.peerMap {
		peerMap[peerIp] = count
	}

	return peerMap
}

func (c *Cache) GetPeers() []string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	sortArray := make([]schema.PeerCount, 0)
	for peerIp, count := range c.peerMap {
		sortArray = append(sortArray, schema.PeerCount{
			Peer:  peerIp,
			Count: count,
		})
	}
	sort.Slice(sortArray, func(i, j int) bool {
		return sortArray[i].Count > sortArray[j].Count
	})
	peers := make([]string, 0)
	for _, peerCnt := range sortArray {
		peers = append(peers, peerCnt.Peer)
	}
	// add arweave.net gateway
	return append([]string{"arweave.net"}, peers...)
}

func (c *Cache) UpdatePeers(peerMap map[string]int64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.peerMap = peerMap
}

func (c *Cache) GetConstTx() *types.Transaction {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.constTx
}

func fetchAnchor(arCli *goar.Client, peers []string) (string, error) {
	anchor, err := arCli.GetTransactionAnchor()
	if err != nil {
		pNode := goar.NewTempConn()
		for _, peer := range peers {
			pNode.SetTempConnUrl("http://" + peer)
			pNode.SetTimeout(time.Second * 10)
			anchor, err = pNode.GetTransactionAnchor()
			if err == nil && len(anchor) > 0 {
				break
			}
		}
	}
	return anchor, err
}

func fetchArInfo(arCli *goar.Client, peers []string) (*types.NetworkInfo, error) {
	info, err := arCli.GetInfo()
	if err != nil {
		pNode := goar.NewTempConn()
		for _, peer := range peers {
			pNode.SetTempConnUrl("http://" + peer)
			pNode.SetTimeout(time.Second * 10)
			info, err = pNode.GetInfo()
			if err == nil && info != nil {
				break
			}
		}
	}
	return info, err
}

func fetchArFee(arCli *goar.Client, peers []string) (schema.ArFee, error) {
	// base fee /fee/0  datasize = 0,data = nil
	var basePrice, deltaPrice int64
	var err1, err2 error
	basePrice, err1 = arCli.GetTransactionPrice(0, nil)
	deltaPrice, err2 = arCli.GetTransactionPrice(1, nil)
	if err1 != nil || err2 != nil {
		pNode := goar.NewTempConn()
		for _, peer := range peers {
			pNode.SetTempConnUrl("http://" + peer)
			pNode.SetTimeout(time.Second * 10)
			basePrice, err1 = pNode.GetTransactionPrice(0, nil)
			deltaPrice, err2 = pNode.GetTransactionPrice(1, nil)
			if err1 == nil && err2 == nil { // fetch fee from one peer
				break
			}
		}
	}
	if err1 != nil || err2 != nil {
		return schema.ArFee{}, errors.New("fetch fee failed")
	}
	return schema.ArFee{Base: basePrice, PerChunk: deltaPrice - basePrice}, nil
}
