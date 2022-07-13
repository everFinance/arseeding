package arseeding

import (
	"github.com/everFinance/goar/utils"
	"github.com/prometheus/client_golang/prometheus"
	"math/big"
)

const (
	MetricNameSpace = "arseeding"
)

var (
	bundlerBalance = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: MetricNameSpace,
			Name:      "bundler_balance",
			Help:      "upload data to arweave",
		},
		[]string{"bundler", "token"},
	)
)

func init() {
	prometheus.MustRegister(
		bundlerBalance,
	)
}

func metricBundlerBalance(bal *big.Int, addr string) {
	arAmount, _ := utils.WinstonToAR(bal).Float64()
	bundlerBalance.WithLabelValues(addr, "AR").Set(arAmount)
}
