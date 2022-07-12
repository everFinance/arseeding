package config

func (c *Config) runJobs() {
	// 定时更新 config 到缓存
	c.scheduler.Every(1).Minute().SingletonMode().Do(c.updateFee)
	c.scheduler.Every(1).Minute().SingletonMode().Do(c.updateIPWhiteList)
	c.scheduler.Every(1).Minute().SingletonMode().Do(c.updateApiKeys)

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

func (c *Config) updateApiKeys() {
	keys, err := c.wdb.GetAllApiKeys()
	if err != nil {
		return
	}
	apiKeys := make(map[string]int64, 0)
	for _, key := range keys {
		if key.Available {
			apiKeys[key.Key] = key.Level
		}
	}
	c.apiKeys = apiKeys
}
