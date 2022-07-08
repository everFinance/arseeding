package config

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/everFinance/arseeding/config/schema"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Wdb struct {
	Db *gorm.DB
}

func NewWdb(dsn string) *Wdb {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:          logger.Default.LogMode(logger.Error),
		CreateBatchSize: 10,
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
	if err == gorm.ErrRecordNotFound {
		fee = schema.FeeConfig{
			SpeedTxFee:     0,
			BundleServeFee: 0,
		}
		return fee, nil
	}
	return
}

func (w *Wdb) Close() {
	sql, err := w.Db.DB()
	if err == nil {
		sql.Close()
	}
}
