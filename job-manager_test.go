package seeding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterAndRemoveJob(t *testing.T) {
	manager := NewJobManager(3)

	_, err := manager.RegisterJob("1", jobTypeBroadcast, 300)
	assert.NoError(t, err)

	_, err = manager.RegisterJob("2", jobTypeBroadcast, 300)
	assert.NoError(t, err)

	id3, err := manager.RegisterJob("3", jobTypeSync, 300)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(manager.status))

	id, err := manager.RegisterJob("3", jobTypeSync, 300)
	assert.Equal(t, "", id)
	assert.Equal(t, "fully loaded", err.Error())

	// remove
	manager.RemoveJob(id3)
	assert.Equal(t, 2, len(manager.status))
}

func TestInc(t *testing.T) {
	manager := NewJobManager(3)

	id, err := manager.RegisterJob("1", jobTypeBroadcast, 300)
	assert.NoError(t, err)

	manager.IncSuccessed(id)
	jobs := manager.GetJobs()
	assert.Equal(t, int64(1), jobs[id].CountSuccessed)
	assert.Equal(t, int64(0), jobs[id].CountFailed)
	assert.Equal(t, int64(0), jobs[id].NumOfProcessed)

	manager.IncFalied(id)
	jobs = manager.GetJobs()
	assert.Equal(t, int64(1), jobs[id].CountSuccessed)
	assert.Equal(t, int64(1), jobs[id].CountFailed)
	assert.Equal(t, int64(0), jobs[id].NumOfProcessed)

	manager.IncProcessed(id)
	jobs = manager.GetJobs()
	assert.Equal(t, int64(1), jobs[id].CountSuccessed)
	assert.Equal(t, int64(1), jobs[id].CountFailed)
	assert.Equal(t, int64(1), jobs[id].NumOfProcessed)
}
