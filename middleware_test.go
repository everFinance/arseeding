package arseeding

import (
	"github.com/everFinance/goarns"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSandboxMiddleware(t *testing.T) {
	host := "p6qmubetdqoqlsoktncg3hiec2nbyjmgqgmhboopftn67xfk.arseed.web3infura.io"

	res := getRequestSandbox(host)
	t.Log(res)
}

func TestGetSubDomain(t *testing.T) {
	host := "cookbook.arseed.web3infura.io"

	res := getSubDomain(host)
	t.Log(res)
}

func TestLimiterMiddleware(t *testing.T) {
	dreUrl := "https://dre-1.warp.cc"
	arNSAddress := "bLAgYxAdX2Ry-nt6aH2ixgvJXbpsEYm28NgJgyqfs-U"
	timeout := 10 * time.Second

	a := goarns.NewArNS(dreUrl, arNSAddress, timeout)

	domain := "cookbook"
	txId, err := a.QueryLatestRecord(domain)
	assert.NoError(t, err)
	t.Log(txId)
}
