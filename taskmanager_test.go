package arseeding

import (
	"github.com/everFinance/arseeding/schema"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddDelTask(t *testing.T) {
	tkm := NewTaskMg()
	arId := "O3VwBusl0PNKusWcDF44uPt-sNuhywgeKxOmQpDqGc0"
	tkType := schema.TaskTypeSync
	tkm.AddTask(arId, tkType)
	ok := tkm.exist(arId, tkType)
	assert.Equal(t, true, ok)

	tkm.DelTask(arId, tkType)
	ok = tkm.exist(arId, tkType)
	assert.Equal(t, false, ok)

}
