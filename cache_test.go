package arseeding

import (
	"github.com/everFinance/goar"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCache_GetPeers(t *testing.T) {
	url := "https://arweave.net"
	cli := goar.NewClient(url)
	// key->val ,node->weight
	peerMap := map[string]int64{
		"node1": 20,
		"node2": 10,
		"node3": 2,
		"node4": 31,
	}
	c := NewCache(cli, peerMap)
	// GetPeers always contain "arweave.net" at index0 and sort nodes by weight
	expectPeers := []string{"arweave.net", "node4", "node1", "node2", "node3"}
	peers := c.GetPeers()
	assert.Equal(t, expectPeers, peers)
}
