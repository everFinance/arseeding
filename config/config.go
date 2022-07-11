package config

import (
	"github.com/go-co-op/gocron"
	"time"
)

type Config struct {
	wdb            *Wdb
	speedTxFee     int64
	bundleServeFee int64
	scheduler      *gocron.Scheduler
}

func New(configDSN string) *Config {
	wdb := NewWdb(configDSN)
	err := wdb.Migrate()
	if err != nil {
		panic(err)
	}
	fee, err := wdb.GetFee()
	if err != nil {
		panic(err)
	}
	return &Config{
		wdb:            wdb,
		speedTxFee:     fee.SpeedTxFee,
		bundleServeFee: fee.BundleServeFee,
		scheduler:      gocron.NewScheduler(time.UTC),
	}
}

func (c *Config) GetSpeedFee() int64 {
	return c.speedTxFee
}

func (c *Config) GetServeFee() int64 {
	return c.bundleServeFee
}

func (c *Config) Run() {
	go c.runJobs()
}

func (c *Config) Close() {
	c.wdb.Close()
}
