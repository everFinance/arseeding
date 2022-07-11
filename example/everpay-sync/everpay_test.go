package everpay_sync

import (
	"testing"
)

func Test_EverPaySync(t *testing.T) {
	dsn := "root@tcp(127.0.0.1:3306)/sandy_test?charset=utf8mb4&parseTime=True&loc=Local"
	seedUrl := "https://arseed.web3infura.io" // your deployed arseeding services
	epSync := New(dsn, seedUrl)
	epSync.Run()

	select {}
}
