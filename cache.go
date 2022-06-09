package arseeding

import (
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar/types"
	"sync"
)

type Cache struct {
	arInfo *types.NetworkInfo
	anchor string
	price  schema.TxPrice
	lock   sync.RWMutex
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

func (c *Cache) GetPrice() schema.TxPrice {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.price
}

func (c *Cache) UpdatePrice(price schema.TxPrice) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.price = price
}
