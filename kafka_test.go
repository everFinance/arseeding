package arseeding

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewKWriter(t *testing.T) {
	uri := "34.220.174.25:9092"
	topic := BlockTopic
	w, err := NewKWriter(topic, uri)
	assert.NoError(t, err)
	err = w.Write([]byte("sandy test"))
	assert.NoError(t, err)
}
