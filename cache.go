package arseeding

import (
	"errors"
	"github.com/everFinance/goar/types"
	"sync"
)

type Cache struct {
	arweaveInfo   *types.NetworkInfo
	anchor        string
	basePrice     int64
	perChunkPrice int64
	lock          sync.RWMutex
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

func (c *Cache) GetPrice() (int64, int64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if c.basePrice == 0 || c.perChunkPrice == 0 {
		return 0, 0, ErrNotExist
	}
	return c.basePrice, c.perChunkPrice, nil
}

func (c *Cache) UpdateBasePrice(price int64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.basePrice = price
}

func (c *Cache) UpdateDeltaPrice(price int64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.perChunkPrice = price
}
