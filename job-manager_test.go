package seeding

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRegisterAndRemoveJob(t *testing.T) {
	manager := NewJobManager(3)

	err := manager.RegisterJob("1", jobTypeBroadcast, 300)
	assert.NoError(t, err)

	err = manager.RegisterJob("2", jobTypeBroadcast, 300)
	assert.NoError(t, err)

	err = manager.RegisterJob("3", jobTypeSync, 300)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(manager.status))

	err = manager.RegisterJob("3", jobTypeSync, 300)
	assert.Equal(t, "exist job", err.Error())

	err = manager.RegisterJob("4", jobTypeSync, 300)
	assert.Equal(t, "fully loaded", err.Error())

	// remove
	manager.UnregisterJob("3", jobTypeSync)
	assert.Equal(t, 2, len(manager.status))
}

func TestInc(t *testing.T) {
	manager := NewJobManager(3)

	arId := "1"
	id := AssembleId(arId, jobTypeBroadcast)
	err := manager.RegisterJob(arId, jobTypeBroadcast, 300)
	assert.NoError(t, err)

	manager.IncSuccessed(arId, jobTypeBroadcast)
	jobs := manager.GetJobs()
	assert.Equal(t, int64(1), jobs[id].CountSuccessed)
	assert.Equal(t, int64(0), jobs[id].CountFailed)

	manager.IncFailed(arId, jobTypeBroadcast)
	jobs = manager.GetJobs()
	assert.Equal(t, int64(1), jobs[id].CountSuccessed)
	assert.Equal(t, int64(1), jobs[id].CountFailed)
}
