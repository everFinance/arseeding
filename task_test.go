package arseeding

import (
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestTasks(t *testing.T) {
	arId := "O3VwBusl0PNKusWcDF44uPt-sNuhywgeKxOmQpDqGc0"
	tkm := NewTaskMg()
	tk := &schema.Task{}
	tkm.taskMap = map[string]*schema.Task{
		schema.TaskTypeSync + "-" + arId:          tk,
		schema.TaskTypeBroadcast + "-" + arId:     tk,
		schema.TaskTypeBroadcastMeta + "-" + arId: tk,
	}
	dbPath := "./data/tmp.db"
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	cli := goar.NewClient("https://arweave.net")
	aa := &Arseeding{
		store:  s,
		taskMg: tkm,
		arCli:  cli,
		cache:  NewCache(cli, nil),
	}
	err = aa.syncTask(arId)
	assert.NoError(t, err)

	err = aa.broadcastTxMetaTask(arId)
	assert.NoError(t, err)

	err = aa.broadcastTxTask(arId)
	assert.NoError(t, err)

	err = aa.setProcessedTask(arId, "sync")
	assert.NoError(t, err)

	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)

}
