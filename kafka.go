package arseeding

import (
	"context"
	"github.com/segmentio/kafka-go"
)

const (
	partition  = 0
	ItemTopic  = "arseeding_transaction"
	BlockTopic = "arseeding_block"
)

type KWriter struct {
	topic string
	conn  *kafka.Conn
}

func NewKWriter(topic string, uri string) (*KWriter, error) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", uri, topic, partition)
	if err != nil {
		return nil, err
	}

	return &KWriter{
		topic: topic,
		conn:  conn,
	}, nil
}

func (kw *KWriter) Write(body []byte) error {
	_, err := kw.conn.Write(body)
	return err
}

func (kw *KWriter) Close() {
	kw.conn.Close()
}

func NewKWriters(uri string) (map[string]*KWriter, error) {
	itemWriter, err := NewKWriter(ItemTopic, uri)
	if err != nil {
		return nil, err
	}
	blockWriter, err := NewKWriter(BlockTopic, uri)
	if err != nil {
		return nil, err
	}
	return map[string]*KWriter{
		ItemTopic:  itemWriter,
		BlockTopic: blockWriter,
	}, nil
}
