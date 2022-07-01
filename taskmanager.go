package arseeding

import (
	"errors"
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"strings"
	"sync"
	"time"
)

type TaskManager struct {
	taskMap map[string]*schema.Task // key: taskType-arId
	tkChan  chan string             // val: taskType-arId
	locker  sync.RWMutex
}

func NewTaskMg() *TaskManager {
	return &TaskManager{
		taskMap: make(map[string]*schema.Task),
		tkChan:  make(chan string),
		locker:  sync.RWMutex{},
	}
}

func (m *TaskManager) InitTaskMg(boltDb *Store) error {
	taskIds, err := boltDb.LoadAllPendingTaskIds()
	if err != nil {
		return err
	}

	for _, tkId := range taskIds {
		arId, tkType, err := splitTaskId(tkId)
		if err != nil {
			log.Error("splitTaskId", "err", err, "tkId", tkId)
			continue
		}
		m.AddTask(arId, tkType)
		go m.PutToTkChan(arId, tkType)
	}

	return nil
}

func (m *TaskManager) AddTask(arid, taskType string) {
	m.locker.Lock()
	defer m.locker.Unlock()

	id := assembleTaskId(arid, taskType)
	_, ok := m.taskMap[id]
	if ok {
		return
	}

	m.taskMap[id] = &schema.Task{
		ArId:     arid,
		TaskType: taskType,
	}
}

func (m *TaskManager) DelTask(arid, taskType string) {
	m.locker.Lock()
	defer m.locker.Unlock()

	id := assembleTaskId(arid, taskType)
	delete(m.taskMap, id)
}

func (m *TaskManager) IncSuccessed(arid, taskType string) {
	m.locker.Lock()
	defer m.locker.Unlock()
	id := assembleTaskId(arid, taskType)
	if s, ok := m.taskMap[id]; ok {
		s.CountSuccessed += 1
	}
}

func (m *TaskManager) IncFailed(arid, taskType string) {
	m.locker.Lock()
	defer m.locker.Unlock()
	id := assembleTaskId(arid, taskType)
	if s, ok := m.taskMap[id]; ok {
		s.CountFailed += 1
	}
}

func (m *TaskManager) TaskBeginSet(arid, taskType string, totalPeer int) error {
	m.locker.RLock()
	defer m.locker.RUnlock()
	id := assembleTaskId(arid, taskType)
	tk, ok := m.taskMap[id]
	if !ok {
		return schema.ErrNotFound
	}
	tk.Timestamp = time.Now().Unix()
	tk.TotalPeer = totalPeer
	return nil
}

func (m *TaskManager) GetTask(arid, taskType string) *schema.Task {
	m.locker.RLock()
	defer m.locker.RUnlock()
	id := assembleTaskId(arid, taskType)
	tk := schema.Task{}
	j, ok := m.taskMap[id]
	if ok {
		tk = *j
		return &tk
	}
	return nil
}

func (m *TaskManager) CloseTask(arid, taskType string) error {
	m.locker.RLock()
	defer m.locker.RUnlock()
	id := assembleTaskId(arid, taskType)
	tk, ok := m.taskMap[id]
	if ok {
		tk.Close = true
	} else {
		return schema.ErrNotFound
	}
	return nil
}

func (m *TaskManager) IsClosed(arid, taskType string) bool {
	m.locker.RLock()
	defer m.locker.RUnlock()
	id := assembleTaskId(arid, taskType)
	tk, ok := m.taskMap[id]
	if ok {
		return tk.Close
	}
	return false
}

func (m *TaskManager) GetTasks() (tasks map[string]schema.Task) {
	m.locker.RLock()
	defer m.locker.RUnlock()

	tasks = make(map[string]schema.Task, len(m.taskMap))
	for id, tk := range m.taskMap {
		tasks[id] = *tk
	}
	return
}

func (m *TaskManager) GetUnconfirmedTxFromPeers(arId, taskType string, peers []string) (*types.Transaction, error) {
	pNode := goar.NewTempConn()
	for _, peer := range peers {
		if m.IsClosed(arId, taskType) {
			return nil, schema.ErrTaskClosed
		}

		pNode.SetTempConnUrl("http://" + peer)
		tx, err := pNode.GetUnconfirmedTx(arId)
		if err != nil {
			continue
		}
		return tx, nil
	}

	return nil, fmt.Errorf("get unconfirmed tx failed; arId: %s", arId)
}

func (m *TaskManager) GetTxDataFromPeers(arId, taskType string, peers []string) ([]byte, error) {
	pNode := goar.NewTempConn()
	for _, peer := range peers {
		if m.IsClosed(arId, taskType) {
			return nil, schema.ErrTaskClosed
		}
		pNode.SetTempConnUrl("http://" + peer)
		data, err := pNode.DownloadChunkData(arId)
		if err != nil {
			m.IncFailed(arId, taskType)
			continue
		}
		m.IncSuccessed(arId, taskType)
		return data, nil
	}

	return nil, schema.ErrFetchData
}

func (m *TaskManager) BroadcastTxMeta(arId, taskType string, tx *types.Transaction, peers []string) {
	pNode := goar.NewTempConn()
	pNode.SetTimeout(10 * time.Second)
	for _, peer := range peers {
		pNode.SetTempConnUrl("http://" + peer)
		status, code, _ := pNode.SubmitTransaction(&types.Transaction{
			Format:    tx.Format,
			ID:        tx.ID,
			LastTx:    tx.LastTx,
			Owner:     tx.Owner,
			Tags:      tx.Tags,
			Target:    tx.Target,
			Quantity:  tx.Quantity,
			Data:      "",
			DataSize:  tx.DataSize,
			DataRoot:  tx.DataRoot,
			Reward:    tx.Reward,
			Signature: tx.Signature,
		})

		if code != 200 && code != 208 && !strings.Contains(status, "Transaction is already in the mempool.") {
			m.IncFailed(arId, taskType)
		} else {
			// success send
			m.IncSuccessed(arId, taskType)
		}

		// listen close taskMap
		if m.IsClosed(arId, taskType) {
			log.Debug("task is closed", "arId", arId, "taskType", taskType)
			return
		}
	}
	// close job
	m.CloseTask(arId, taskType)
}

func (m *TaskManager) BroadcastData(arId, taskType string, tx *types.Transaction, peers []string, txPosted bool) {
	pNode := goar.NewTempConn()
	for _, peer := range peers {
		pNode.SetTempConnUrl("http://" + peer)
		uploader, err := goar.CreateUploader(pNode, tx, nil)
		if err != nil {
			m.IncFailed(arId, taskType)
			continue
		}

		// post tx
		if !txPosted {
			pNode.SubmitTransaction(&types.Transaction{
				Format:    tx.Format,
				ID:        tx.ID,
				LastTx:    tx.LastTx,
				Owner:     tx.Owner,
				Tags:      tx.Tags,
				Target:    tx.Target,
				Quantity:  tx.Quantity,
				Data:      "",
				DataSize:  tx.DataSize,
				DataRoot:  tx.DataRoot,
				Reward:    tx.Reward,
				Signature: tx.Signature,
			})
		}

		uploader.TxPosted = true // only broadcast tx data
		if err = uploader.Once(); err != nil {
			m.IncFailed(arId, taskType)
		} else {
			// success send
			m.IncSuccessed(arId, taskType)
		}

		// listen close taskMap
		if m.IsClosed(arId, taskType) {
			return
		}
	}
	m.CloseTask(arId, taskType)
	return
}

func (m *TaskManager) PopTkChan() <-chan string {
	return m.tkChan
}

func (m *TaskManager) PutToTkChan(arId, taskType string) {
	m.tkChan <- assembleTaskId(arId, taskType)
}

func assembleTaskId(arid, taskType string) string {
	return strings.ToLower(taskType) + "-" + arid
}

func splitTaskId(taskId string) (arid, taskType string, err error) {
	ss := strings.SplitN(taskId, "-", 2)
	if len(ss) != 2 {
		err = errors.New("taskId incorrect")
		return
	}
	taskType, arid = ss[0], ss[1]
	return
}
