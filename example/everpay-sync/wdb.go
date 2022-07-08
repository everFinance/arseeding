package everpay_sync

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type RollupArId struct {
	gorm.Model
	ArId string `gorm:"index:idx01"`
	Post bool
}

func (r RollupArId) TableName() string {
	return "rollup_ar_id_prod"
}

type Wdb struct {
	Db *gorm.DB
}

func NewWdb(dsn string) *Wdb {
	logLevel := logger.Info
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:          logger.Default.LogMode(logLevel),
		CreateBatchSize: 200,
	})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&RollupArId{})

	log.Info("connect wdb success")
	return &Wdb{Db: db}
}

func (w *Wdb) Insert(arId string) error {
	return w.Db.Create(&RollupArId{ArId: arId}).Error
}

func (w *Wdb) UpdatePosted(arId string) error {
	return w.Db.Model(&RollupArId{}).Where("ar_id = ?", arId).Update("post", true).Error
}

func (w *Wdb) GetArIds(fromId int) ([]RollupArId, error) {
	rollupTxs := make([]RollupArId, 0)
	err := w.Db.Model(&RollupArId{}).Where("id > ?", fromId).Limit(50).Find(&rollupTxs).Error
	return rollupTxs, err
}

func (w *Wdb) GetLastPostTx() (RollupArId, error) {
	tx := RollupArId{}
	err := w.Db.Model(&RollupArId{}).Order("id desc").Limit(1).Scan(&tx).Error
	if err == gorm.ErrRecordNotFound {
		return tx, nil
	}
	return tx, err
}

func (w *Wdb) GetNeedPostTxs() ([]RollupArId, error) {
	txs := make([]RollupArId, 0)
	err := w.Db.Model(&RollupArId{}).Where("post = ?", false).Limit(100).Find(&txs).Error
	return txs, err
}
