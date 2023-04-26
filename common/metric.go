package common

import (
	"github.com/everFinance/go-everpay/common"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var log = common.NewLog("common")

func NewMetricServer() {
	port := ":9000"
	log.Info("Starting metric server", "listen", port)
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(port, nil); err != nil {
			panic(err)
		}
	}()
}
