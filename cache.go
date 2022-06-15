package arseeding

import (
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"sync"
)

type Cache struct {
	arInfo types.NetworkInfo
	anchor string
	fee    schema.ArFee
	peers  []string
	lock   sync.RWMutex
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

func fetchAnchor(arCli *goar.Client, peers []string) (string, error) {
	anchor, err := arCli.GetTransactionAnchor()
	if err != nil {
		pNode := goar.NewTempConn()
		for _, peer := range peers {
			pNode.SetTempConnUrl("http://" + peer)
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

	littleData := make([]byte, 1)
	basePrice, err1 = arCli.GetTransactionPrice(nil, nil)
	deltaPrice, err2 = arCli.GetTransactionPrice(littleData, nil)
	if err1 != nil || err2 != nil {
		pNode := goar.NewTempConn()
		for _, peer := range peers {
			pNode.SetTempConnUrl("http://" + peer)
			basePrice, err1 = pNode.GetTransactionPrice(nil, nil)
			deltaPrice, err2 = pNode.GetTransactionPrice(littleData, nil)
			if err1 == nil && err2 == nil { // fetch fee from one peer
				break
			}
		}
	}
	if err1 != nil || err2 != nil {
		return schema.ArFee{}, ErrFetchArFee
	}
	return schema.ArFee{Base: basePrice, PerChunk: deltaPrice - basePrice}, nil
}
