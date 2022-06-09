package arseeding

import (
	"errors"
	"github.com/everFinance/goar/types"
	"sync"
)

type Cache struct {
	arweaveInfo *types.NetworkInfo
	anchor      string
	price       TxPrice
	lock        sync.RWMutex
}

type TxPrice struct {
	basePrice     int64
	perChunkPrice int64
}

var (
	ErrPriceType = errors.New("price type err")
)

func (c *Cache) GetInfo() (*types.NetworkInfo, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	info := c.arweaveInfo
	if info == nil {
		return info, ErrNotExist
	}
	return info, nil
}

func (c *Cache) UpdateInfo(info *types.NetworkInfo) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.arweaveInfo = info
}

func (c *Cache) GetAnchor() (string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	anchor := c.anchor
	if anchor == "" {
		return "", ErrNotExist
	}
	return anchor, nil
}

func (c *Cache) UpdateAnchor(anchor string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.anchor = anchor
}

func (c *Cache) GetPrice() (*TxPrice, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if c.price.basePrice == 0 {
		return nil, ErrNotExist
	}
	return &c.price, nil
}

func (c *Cache) UpdatePrice(price TxPrice) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.price = price
}
