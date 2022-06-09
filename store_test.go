package arseeding

import (
	"encoding/json"
	"testing"
)

func TestMarshal(t *testing.T) {
	peers := []string{"p1", "p2", "p3"}
	pByte, err := json.Marshal(peers)
	if err != nil {
		t.Logf("%v", err)
	}
	newPeers := make([]string, 0)
	err = json.Unmarshal(pByte, &newPeers)
	if err != nil {
		t.Logf("%v", err)
	}
	t.Logf("%v", newPeers[1] == peers[1])
}
