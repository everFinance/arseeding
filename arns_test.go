package arseeding

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestArNs(t *testing.T) {
	cuUrl := "https://cu.ar-io.dev"
	muUrl := ""
	arns := NewArNs(cuUrl, muUrl)

	domain := "permaswap"
	record, err := arns.GetRootDomainRecord(domain)
	assert.NoError(t, err)
	t.Log(*record)

	state, err := arns.GetDomainProcessState(record.ProcessId)
	assert.NoError(t, err)
	t.Log(*state)
}

func TestArNs_GetDomainRecord(t *testing.T) {
	cuUrl := "https://cu.ar-io.dev"
	muUrl := ""
	arns := NewArNs(cuUrl, muUrl)

	domain := "ao"
	txId, err := arns.QueryDomainTxId(domain)
	assert.NoError(t, err)
	t.Log(txId)
}
