package arseeding

import (
	"github.com/everFinance/goar/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"math/big"
	"net/http"
)

func NewMetricServer(serverName string) {
	port := ":9000"
	log.Info("Starting metric server", "listen", port)
	http.Handle("/metrics", promhttp.Handler())
	registerApiMetrics(serverName)
	go func() {
		if err := http.ListenAndServe(port, nil); err != nil {
			panic(err)
		}
	}()
}

var (
	bundlerBalance *prometheus.GaugeVec
)

func registerApiMetrics(serverName string) {
	bundlerBalance = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: serverName,
			Name:      "bundler_balance",
			Help:      "upload data to arweave",
		},
		[]string{"bundler", "token"},
	)

	prometheus.MustRegister(
		bundlerBalance,
	)
}

func UpdateBalance(bal *big.Int, addr string) {
	arAmount, _ := utils.WinstonToAR(bal).Float64()
	bundlerBalance.WithLabelValues(addr, "AR").Set(arAmount)
}
