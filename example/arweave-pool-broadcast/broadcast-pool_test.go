package arweave_pool_broadcast

import "testing"

func TestBcPool_Run(t *testing.T) {
	seedUrl := "http://127.0.0.1:8080"
	s := New(seedUrl)
	s.Run()

	select {}
}
