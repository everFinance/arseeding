package config

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/everFinance/arseeding/config/schema"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
)

type Wdb struct {
	Db *gorm.DB
}

func NewWdb(dsn string) *Wdb {
	runEnv := os.Getenv("RUN_ENV")
	logLevel := logger.Warn
	if runEnv == "dev" || runEnv == "" {
		logLevel = logger.Info
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:          logger.Default.LogMode(logLevel), // 日志 level 设置, prod 使用 warn
		CreateBatchSize: 200,
	})
	if err != nil {
		panic(err)
	}
	log.Info("connect config db success")
	return &Wdb{Db: db}
}

func (w *Wdb) Migrate() error {
	return w.Db.AutoMigrate(&schema.FeeConfig{})
}

func (w *Wdb) GetFee() (fee schema.FeeConfig, err error) {
	err = w.Db.First(&fee).Error
	return
}

func (w *Wdb) Close() {
	sql, err := w.Db.DB()
	if err == nil {
		sql.Close()
	}
}
