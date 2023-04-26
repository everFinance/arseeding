package config

import (
	"github.com/everFinance/arseeding/config/schema"
	"github.com/everFinance/go-everpay/common"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"path"
)

const (
	sqliteName = "seed.sqlite"
)

var log = common.NewLog("config")

type Wdb struct {
	Db *gorm.DB
}

func NewMysqlDb(dsn string) *Wdb {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:          logger.Default.LogMode(logger.Silent),
		CreateBatchSize: 200,
	})
	if err != nil {
		panic(err)
	}
	log.Info("connect mysql db success")
	return &Wdb{Db: db}
}

func NewSqliteDb(dbDir string) *Wdb {
	if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
		panic(err)
	}
	db, err := gorm.Open(sqlite.Open(path.Join(dbDir, sqliteName)), &gorm.Config{
		Logger:          logger.Default.LogMode(logger.Silent),
		CreateBatchSize: 200,
	})
	if err != nil {
		panic(err)
	}
	log.Info("connect sqlite db success")
	return &Wdb{Db: db}

}

func (w *Wdb) Migrate() error {
	return w.Db.AutoMigrate(&schema.FeeConfig{},
		&schema.IpRateWhitelist{},
		&schema.Param{})
}

func (w *Wdb) Close() {
	sql, err := w.Db.DB()
	if err == nil {
		sql.Close()
	}
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

func (w *Wdb) GetParam() (param schema.Param, err error) {
	err = w.Db.First(&param).Error
	if err == gorm.ErrRecordNotFound {
		param = schema.Param{
			ChunkConcurrentNum: 0,
		}
		return param, nil
	}
	return
}
