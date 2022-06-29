package arseeding

import (
	"github.com/everFinance/arseeding/schema"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"math"
	"time"
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

func (w *Wdb) Migrate(noFee bool) error {
	err := w.Db.AutoMigrate(&schema.Order{}, &schema.OnChainTx{})
	if err != nil {
		return err
	}
	if !noFee {
		err = w.Db.AutoMigrate(&schema.TokenPrice{}, &schema.ReceiptEverTx{})
	}
	return err
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

func (w *Wdb) GetExpiredOrders() ([]schema.Order, error) {
	now := time.Now().Unix()
	ords := make([]schema.Order, 0, 10)
	err := w.Db.Model(&schema.Order{}).Where("payment_status = ? and payment_expired_time < ?", schema.UnPayment, now).Find(&ords).Error
	return ords, err
}

func (w *Wdb) UpdateOrdToExpiredStatus(id uint) error {
	data := make(map[string]interface{})
	data["payment_status"] = schema.ExpiredPayment
	data["on_chain_status"] = schema.FailedOnChain
	return w.Db.Model(&schema.Order{}).Where("id = ?", id).Updates(data).Error
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

func (w *Wdb) GetNeedOnChainOrders() ([]schema.Order, error) {
	res := make([]schema.Order, 0)
	err := w.Db.Where("payment_status = ?  and on_chain_status = ?", schema.SuccPayment, schema.WaitOnChain).Find(&res).Error
	return res, err
}

func (w *Wdb) UpdateOrdOnChainStatus(itemId, status string, tx *gorm.DB) error {
	db := w.Db
	if tx != nil {
		db = tx
	}
	return db.Model(&schema.Order{}).Where("item_id = ?", itemId).Update("on_chain_status", status).Error
}

func (w *Wdb) GetOrdersBySigner(signer string, cursorId int64, num int) ([]schema.Order, error) {
	if cursorId <= 0 {
		cursorId = math.MaxInt64
	}
	records := make([]schema.Order, 0, num)
	err := w.Db.Where("signer = ? and id < ?", signer, cursorId).Order("id DESC").Limit(num).Find(&records).Error
	return records, err
}

func (w *Wdb) ExistProcessedOrderItem(itemId string) (res schema.Order, exist bool) {
	err := w.Db.Where("item_id = ? and (on_chain_status = ? or on_chain_status = ?)", itemId, schema.PendingOnChain, schema.SuccOnChain).First(&res).Error
	if err == nil {
		exist = true
	}
	return
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

func (w *Wdb) GetArPrice() (float64, error) {
	res := schema.TokenPrice{}
	err := w.Db.Where("symbol = ?", "AR").First(&res).Error
	return res.Price, err
}

func (w *Wdb) InsertReceiptTx(tx schema.ReceiptEverTx) error {
	return w.Db.Clauses(clause.OnConflict{DoNothing: true}).Create(&tx).Error
}

func (w *Wdb) GetLastEverRawId() (uint64, error) {
	tx := schema.ReceiptEverTx{}
	err := w.Db.Model(&schema.ReceiptEverTx{}).Last(&tx).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	return tx.RawId, err
}

func (w *Wdb) GetReceiptsByStatus(status string) ([]schema.ReceiptEverTx, error) {
	res := make([]schema.ReceiptEverTx, 0)
	err := w.Db.Model(&schema.ReceiptEverTx{}).Where("status = ?", status).Find(&res).Error
	return res, err
}

func (w *Wdb) UpdateReceiptStatus(rawId uint64, status string, tx *gorm.DB) error {
	db := w.Db
	if tx != nil {
		db = tx
	}
	return db.Model(&schema.ReceiptEverTx{}).Where("raw_id = ?", rawId).Update("status", status).Error
}

func (w *Wdb) UpdateRefundErr(rawId uint64, errMsg string) error {
	data := make(map[string]interface{})
	data["status"] = schema.RefundErr
	data["err_msg"] = errMsg
	return w.Db.Model(&schema.ReceiptEverTx{}).Where("raw_id = ?", rawId).Updates(data).Error
}

func (w *Wdb) InsertArTx(tx schema.OnChainTx) error {
	return w.Db.Create(&tx).Error
}

func (w *Wdb) GetArTxByStatus(status string) ([]schema.OnChainTx, error) {
	res := make([]schema.OnChainTx, 0, 10)
	err := w.Db.Where("status = ?", status).Find(&res).Error
	return res, err
}

func (w *Wdb) UpdateArTxStatus(arId, status string, tx *gorm.DB) error {
	db := w.Db
	if tx != nil {
		db = tx
	}
	return db.Model(&schema.OnChainTx{}).Where("ar_id = ?", arId).Update("status", status).Error
}

func (w *Wdb) UpdateArTx(id uint, arId string, curHeight int64, status string) error {
	data := make(map[string]interface{})
	data["ar_id"] = arId
	data["cur_height"] = curHeight
	data["status"] = status
	return w.Db.Model(&schema.OnChainTx{}).Where("id = ?", id).Updates(data).Error
}
