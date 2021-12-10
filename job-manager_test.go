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
	assert.Equal(t, "fully loaded", err.Error())

	// remove
	assert.Equal(t, 2, len(manager.status))
}

func TestInc(t *testing.T) {
	manager := NewJobManager(3)

	arId := "1"
	err := manager.RegisterJob(arId, jobTypeBroadcast, 300)
	assert.NoError(t, err)

	manager.IncSuccessed(arId)
	jobs := manager.GetJobs()
	assert.Equal(t, int64(1), jobs[arId].CountSuccessed)
	assert.Equal(t, int64(0), jobs[arId].CountFailed)
	assert.Equal(t, int64(0), jobs[arId].NumOfProcessed)

	manager.IncFailed(arId)
	jobs = manager.GetJobs()
	assert.Equal(t, int64(1), jobs[arId].CountSuccessed)
	assert.Equal(t, int64(1), jobs[arId].CountFailed)
	assert.Equal(t, int64(0), jobs[arId].NumOfProcessed)

	manager.IncProcessed(arId)
	jobs = manager.GetJobs()
	assert.Equal(t, int64(1), jobs[arId].CountSuccessed)
	assert.Equal(t, int64(1), jobs[arId].CountFailed)
	assert.Equal(t, int64(1), jobs[arId].NumOfProcessed)
}
