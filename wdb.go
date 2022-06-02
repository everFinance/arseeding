package arseeding

import (
	"github.com/everFinance/arseeding/schema"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type Wdb struct {
	Db *gorm.DB
}

func NewWdb(dsn string) *Wdb {
	logLevel := logger.Error
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:          logger.Default.LogMode(logLevel), // 日志 level 设置, prod 使用 warn
		CreateBatchSize: 200,
	})
	if err != nil {
		panic(err)
	}
	log.Info("connect db success")
	return &Wdb{Db: db}
}

func (w *Wdb) Migrate() error {
	return w.Db.AutoMigrate(&schema.Order{}, &schema.TokenPrice{}, &schema.ArFee{}, &schema.ReceiptEverTx{})
}

func (w *Wdb) InsertOrder(order schema.Order) error {
	return w.Db.Create(&order).Error
}

func (w *Wdb) GetUnPaidOrder(signer, currency, fee string) (schema.Order, error) {
	res := schema.Order{}
	err := w.Db.Model(&schema.Order{}).Where("signer = ? and payment_status = ?"+
		" and currency = ? and fee = ?", signer, schema.UnPayment, currency, fee).First(&res).Error
	return res, err
}

func (w *Wdb) UpdateOrderPay(id uint, everHash string, paymentStatus string, tx *gorm.DB) error {
	db := w.Db
	if tx != nil {
		db = tx
	}
	data := make(map[string]interface{})
	data["payment_status"] = paymentStatus
	data["payment_id"] = everHash
	return db.Model(&schema.Order{}).Where("id = ?", id).Updates(data).Error
}

func (w *Wdb) InsertPrices(tps []schema.TokenPrice) error {
	return w.Db.Clauses(clause.OnConflict{DoNothing: true}).Create(&tps).Error
}

func (w *Wdb) UpdatePrice(symbol string, newPrice float64) error {
	return w.Db.Model(&schema.TokenPrice{}).Where("symbol = ?", symbol).Update("price", newPrice).Error
}

func (w *Wdb) GetPrices() ([]schema.TokenPrice, error) {
	res := make([]schema.TokenPrice, 0, 10)
	err := w.Db.Find(&res).Error
	return res, err
}

func (w *Wdb) UpdateArFee(baseFee, perChunkFee int64) error {
	arFee := &schema.ArFee{
		ID:       1,
		Base:     baseFee,
		PerChunk: perChunkFee,
	}
	return w.Db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(arFee).Error
}

func (w *Wdb) GetArFee() (res schema.ArFee, err error) {
	err = w.Db.First(&res).Error
	return
}

func (w *Wdb) InsertReceiptTx(txs []schema.ReceiptEverTx) error {
	return w.Db.Clauses(clause.OnConflict{DoNothing: true}).Create(&txs).Error
}

func (w *Wdb) GetLastPage() (int, error) {
	tx := schema.ReceiptEverTx{}
	err := w.Db.Model(&schema.ReceiptEverTx{}).Order("page desc").Limit(1).Scan(&tx).Error
	if err == gorm.ErrRecordNotFound {
		return 1, nil
	}
	return tx.Page, err
}

func (w *Wdb) GetReceiptsByStatus(status string) ([]schema.ReceiptEverTx, error) {
	res := make([]schema.ReceiptEverTx, 0)
	err := w.Db.Model(&schema.ReceiptEverTx{}).Where("status = ?", status).Find(&res).Error
	return res, err
}

func (w *Wdb) UpdateReceiptStatus(id uint, status string, tx *gorm.DB) error {
	db := w.Db
	if tx != nil {
		db = tx
	}
	return db.Model(&schema.ReceiptEverTx{}).Where("id = ?", id).Update("status", status).Error
}
