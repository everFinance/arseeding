package config

func (c *Config) runJobs() {
	// 定时更新 config 到缓存
	c.scheduler.Every(20).Second().SingletonMode().Do(c.updateFee)

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
