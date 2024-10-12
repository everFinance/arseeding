package sdk

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNew2(t *testing.T) {
	cli := New("https://seed-dev.everpay.io")
	bundler, err := cli.GetBundler()
	assert.NoError(t, err)
	t.Log(bundler)
}

func TestNew(t *testing.T) {
	cli := New("https://arseed.web3infra.dev")
	ss, err := cli.ACli.GetTransactionDataByGateway("hHcZxYgurvoTN-KLylc0QulK9vyaUIHWfqvH7A9anmo")
	assert.NoError(t, err)
	t.Log(string(ss))
}
