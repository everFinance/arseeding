package schema

const (
	TaskTypeBroadcast     = "broadcast"      // include tx and tx data
	TaskTypeBroadcastMeta = "broadcast_meta" //  not include tx data
	TaskTypeSync          = "sync"
	TaskTypeSyncManifest  = "sync_manifest" // sync manifest
)

type Task struct {
	ArId           string `json:"arId"`
	TaskType       string `json:"taskType"`
	CountSuccessed int64  `json:"countSuccessed"`
	CountFailed    int64  `json:"countFailed"`
	TotalPeer      int    `json:"totalPeer"`
	Timestamp      int64  `json:"timestamp"` // begin timestamp
	Close          bool   `json:"close"`
}
