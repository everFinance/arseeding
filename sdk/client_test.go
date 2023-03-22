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
