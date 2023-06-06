package arseeding

import (
	"context"
	"github.com/segmentio/kafka-go"
)

const (
	ItemTopic  = "arseeding_transaction"
	BlockTopic = "arseeding_block"
)

type KWriter struct {
	w *kafka.Writer
}

func NewKWriter(topic string, uri string) (*KWriter, error) {
	w := &kafka.Writer{
		Addr:     kafka.TCP(uri),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &KWriter{
		w: w,
	}, nil
}

func (kw *KWriter) Write(body []byte) error {
	err := kw.w.WriteMessages(
		context.Background(),
		kafka.Message{
			Value: body,
		},
	)
	return err
}

func (kw *KWriter) Close() {
	kw.w.Close()
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
