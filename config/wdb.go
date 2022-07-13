package config

import (
	"github.com/everFinance/arseeding/config/schema"
	"github.com/everFinance/everpay-go/common"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var log = common.NewLog("config")

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
	return w.Db.AutoMigrate(&schema.FeeConfig{}, &schema.IpRateWhitelist{})
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

func (w *Wdb) GetAllAvailableIpRateWhitelist() ([]schema.IpRateWhitelist, error) {
	res := make([]schema.IpRateWhitelist, 0, 10)
	err := w.Db.Where("available = ?", true).Find(&res).Error
	return res, err
}

func (w *Wdb) Close() {
	sql, err := w.Db.DB()
	if err == nil {
		sql.Close()
	}
}
