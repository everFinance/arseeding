package arseeding

import "testing"

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
