package config

func (c *Config) runJobs() {
	c.scheduler.Every(1).Minute().SingletonMode().Do(c.updateFee)
	c.scheduler.Every(1).Minute().SingletonMode().Do(c.updateIPWhiteList)
	c.scheduler.Every(10).Seconds().SingletonMode().Do(c.updateParam)

	c.scheduler.StartAsync()
}

func (c *Config) updateFee() {
	fee, err := c.wdb.GetFee()
	if err != nil {
		return
	}
	c.speedTxFee = fee.SpeedTxFee
	c.bundleServeFee = fee.BundleServeFee
}

func (c *Config) updateIPWhiteList() {
	ips, err := c.wdb.GetAllAvailableIpRateWhitelist()
	if err != nil {
		return
	}
	ipWhiteList := make(map[string]struct{}, 0)
	for _, ip := range ips {
		if ip.Available {
			ipWhiteList[ip.OriginOrIP] = struct{}{}
		}
	}
	c.ipWhiteList = ipWhiteList
}

func (c *Config) updateParam() {
	param, err := c.wdb.GetParam()
	if err != nil {
		return
	}
	c.Param = param
}
