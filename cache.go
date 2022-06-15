package arseeding

import (
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"sync"
	"time"
)

type Cache struct {
	arInfo      *types.NetworkInfo
	anchor      string
	fee         schema.ArFee
	peers       []string
	submitPeers []string           // submit Tx or chunk to those peer
	constTx     *types.Transaction // a legal arTx used to test a peer is available
	lock        sync.RWMutex
}

func (c *Cache) GetInfo() *types.NetworkInfo {
	c.lock.RLock()
	defer c.lock.RUnlock()
	info := c.arInfo
	return info
}

func (c *Cache) UpdateInfo(info *types.NetworkInfo) {
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

func (c *Cache) GetPeers() []string {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.peers
}

func (c *Cache) UpdatePeers(peers []string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.peers = peers
}

func (c *Cache) GetSubmitPeers() []string {
	c.lock.RUnlock()
	defer c.lock.RUnlock()

	return c.submitPeers
}

func (c *Cache) UpdateSubmitPeers(peers []string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.submitPeers = peers
}

func (c *Cache) GetConstTx() *types.Transaction {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.constTx
}

func fetchAnchor(arCli *goar.Client, peers []string) (string, error) {
	anchor, err := arCli.GetTransactionAnchor()
	if err != nil {
		anchorMap := make(map[string]int64, 0)
		var mu sync.Mutex
		pNode := goar.NewTempConn()
		for _, peer := range peers {
			go func(peer string) {
				pNode.SetTempConnUrl("http://" + peer)
				pNode.SetTimeout(time.Second * 10)
				anchor, err := pNode.GetTransactionAnchor()
				if err == nil && len(anchor) > 0 {
					mu.Lock()
					anchorMap[anchor] += 1
					mu.Unlock()
				}
			}(peer)
		}
		time.Sleep(time.Second * 20)
		for anchor, count := range anchorMap {
			if count > 1 {
				return anchor, nil
			}
		}
	}
	return anchor, err
}

func fetchArInfo(arCli *goar.Client, peers []string) (*types.NetworkInfo, error) {
	info, err := arCli.GetInfo()
	if err != nil {
		infos := make([]*types.NetworkInfo, 0)
		var mu sync.Mutex
		pNode := goar.NewTempConn()
		for _, peer := range peers {
			go func(peer string) {
				pNode.SetTempConnUrl("http://" + peer)
				pNode.SetTimeout(time.Second * 10)
				info, err := pNode.GetInfo()
				if err == nil && info != nil {
					mu.Lock()
					infos = append(infos, info)
					mu.Unlock()
				}
			}(peer)
		}
		time.Sleep(time.Second * 20)
		if len(infos) > 0 {
			return infos[0], nil
		}
	}
	return info, err
}

func fetchArFee(arCli *goar.Client, peers []string) (schema.ArFee, error) {
	// base fee /fee/0  datasize = 0,data = nil
	var basePrice, deltaPrice int64
	var err1, err2 error
	fees := make([]schema.ArFee, 0)
	littleData := make([]byte, 1)
	basePrice, err1 = arCli.GetTransactionPrice(nil, nil)
	deltaPrice, err2 = arCli.GetTransactionPrice(littleData, nil)
	if err1 != nil || err2 != nil {
		var mu sync.Mutex
		pNode := goar.NewTempConn()
		for _, peer := range peers {
			go func(peer string) {
				pNode.SetTempConnUrl("http://" + peer)
				pNode.SetTimeout(time.Second * 10)
				basePrice, err1 := pNode.GetTransactionPrice(nil, nil)
				deltaPrice, err2 := pNode.GetTransactionPrice(littleData, nil)
				if err1 == nil && err2 == nil { // fetch fee from one peer
					mu.Lock()
					fees = append(fees, schema.ArFee{Base: basePrice, PerChunk: deltaPrice - basePrice})
					mu.Unlock()
				}
			}(peer)
		}
		time.Sleep(time.Second * 20)
		if len(fees) > 0 {
			return fees[0], nil
		} else {
			return schema.ArFee{}, ErrFetchArFee
		}
	}
	return schema.ArFee{Base: basePrice, PerChunk: deltaPrice - basePrice}, nil
}

func fetchConstTx(arCli *goar.Client, peers []string) (*types.Transaction, error) {
	arId := "e8NNxYmPRVgESMA6cu31OAIe-3wvpWbu9Ng3FhK4FbU"
	tx, err := arCli.GetUnconfirmedTx(arId)
	if err != nil {
		tx, err = arCli.GetUnconfirmedTxFromPeers(arId, peers...)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}
