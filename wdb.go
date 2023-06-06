package arseeding

import (
	"encoding/json"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar/types"
	"gorm.io/datatypes"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"math"
	"os"
	"path"
	"strings"
	"time"
)

const (
	sqliteName = "seed.sqlite"
)

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

// when use sqlite,same index name in different table will lead to migrate failed,

func (w *Wdb) Migrate(noFee, enableManifest bool) error {
	err := w.Db.AutoMigrate(&schema.Order{}, &schema.OnChainTx{}, &schema.AutoApiKey{}, &schema.OrderStatistic{})
	if err != nil {
		return err
	}
	if !noFee {
		err = w.Db.AutoMigrate(&schema.TokenPrice{}, &schema.ReceiptEverTx{})
	}
	if err != nil {
		return err
	}
	if enableManifest {
		err = w.Db.AutoMigrate(&schema.Manifest{})
	}
	return err
}

func (w *Wdb) InsertOrder(order schema.Order) error {
	return w.Db.Create(&order).Error
}

func (w *Wdb) GetUnPaidOrder(itemId string) (schema.Order, error) {
	res := schema.Order{}
	err := w.Db.Model(&schema.Order{}).Where("item_id = ? and payment_status = ?", itemId, schema.UnPayment).Last(&res).Error
	return res, err
}

func (w *Wdb) GetExpiredOrders() ([]schema.Order, error) {
	now := time.Now().Unix()
	ords := make([]schema.Order, 0, 10)
	err := w.Db.Model(&schema.Order{}).Where("payment_status = ? and payment_expired_time < ?", schema.UnPayment, now).Find(&ords).Error
	return ords, err
}
func (w *Wdb) ExistPaidOrd(itemId string) bool {
	ord := &schema.Order{}
	err := w.Db.Model(&schema.Order{}).Where("item_id = ? and payment_status = ?", itemId, schema.SuccPayment).First(ord).Error
	if err == gorm.ErrRecordNotFound {
		return false
	}
	return true
}

func (w *Wdb) IsLatestUnpaidOrd(itemId string, CurExpiredTime int64) bool {
	ord := &schema.Order{}
	err := w.Db.Model(&schema.Order{}).Where("item_id = ? and payment_status = ? and payment_expired_time > ?", itemId, schema.UnPayment, CurExpiredTime).First(ord).Error
	if err == gorm.ErrRecordNotFound {
		return true
	}
	return false
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
	err := w.Db.Model(&schema.Order{}).Where("payment_status = ?  and on_chain_status = ? and sort = ?", schema.SuccPayment, schema.WaitOnChain, false).Limit(2000).Find(&res).Error
	return res, err
}

func (w *Wdb) GetNeedOnChainOrdersSorted() ([]schema.Order, error) {
	res := make([]schema.Order, 0)
	err := w.Db.Model(&schema.Order{}).Where("payment_status = ?  and on_chain_status = ? and sort = ?", schema.SuccPayment, schema.WaitOnChain, true).Limit(2000).Find(&res).Error
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
	err := w.Db.Model(&schema.Order{}).Where("id < ? and signer = ? and on_chain_status != ?", cursorId, signer, schema.FailedOnChain).Order("id DESC").Limit(num).Find(&records).Error
	return records, err
}

func (w *Wdb) GetOrdersByApiKey(apiKey string, cursorId int64, pageSize int, sort string) ([]schema.Order, error) {
	records := make([]schema.Order, 0, pageSize)
	var err error
	if strings.ToUpper(sort) == "ASC" {
		if cursorId <= 0 {
			cursorId = 0
		}
		err = w.Db.Model(&schema.Order{}).Where("api_key = ? and id > ?", apiKey, cursorId).Order("id ASC").Limit(pageSize).Find(&records).Error
	} else {
		if cursorId <= 0 {
			cursorId = math.MaxInt64
		}
		err = w.Db.Model(&schema.Order{}).Where("api_key = ? and id < ?", apiKey, cursorId).Order("id DESC").Limit(pageSize).Find(&records).Error
	}
	return records, err
}

func (w *Wdb) ExistProcessedOrderItem(itemId string) (res schema.Order, exist bool) {
	err := w.Db.Model(&schema.Order{}).Where("item_id = ? and (on_chain_status = ? or on_chain_status = ?)", itemId, schema.PendingOnChain, schema.SuccOnChain).First(&res).Error
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
	timestamp := time.Now().UnixMilli() - 24*60*60*1000 // latest 1 day
	err := w.Db.Model(&schema.ReceiptEverTx{}).Where("status = ? and nonce > ?", status, timestamp).Find(&res).Error
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
	err := w.Db.Model(schema.OnChainTx{}).Where("status = ?", status).Find(&res).Error
	return res, err
}

func (w *Wdb) UpdateArTxStatus(arId, status string, arTxStatus *types.TxStatus, tx *gorm.DB) error {
	db := w.Db
	if tx != nil {
		db = tx
	}
	data := make(map[string]interface{})
	data["status"] = status
	if arTxStatus != nil {
		data["block_id"] = arTxStatus.BlockIndepHash
		data["block_height"] = arTxStatus.BlockHeight
	}
	return db.Model(&schema.OnChainTx{}).Where("ar_id = ?", arId).Updates(data).Error
}

func (w *Wdb) UpdateArTx(id uint, arId string, curHeight int64, dataSize, reward string, status string) error {
	data := make(map[string]interface{})
	data["ar_id"] = arId
	data["cur_height"] = curHeight
	data["data_size"] = dataSize
	data["reward"] = reward
	data["status"] = status
	return w.Db.Model(&schema.OnChainTx{}).Where("id = ?", id).Updates(data).Error
}

func (w *Wdb) GetKafkaOnChains() ([]schema.OnChainTx, error) {
	results := make([]schema.OnChainTx, 0)
	err := w.Db.Model(&schema.OnChainTx{}).Where("block_height > ? and kafka = ? and status = ?", 1188855, false, schema.SuccOnChain).Limit(10).Find(&results).Error
	return results, err
}

func (w *Wdb) KafkaOnChainDone(id uint) error {
	return w.Db.Model(&schema.OnChainTx{}).Where("id = ?", id).Update("kafka", true).Error
}

func (w *Wdb) InsertManifest(mf schema.Manifest) error {
	return w.Db.Create(&mf).Error
}

func (w *Wdb) GetManifestId(mfUrl string) (string, error) {
	res := schema.Manifest{}
	err := w.Db.Model(&schema.Manifest{}).Where("manifest_url = ?", mfUrl).Last(&res).Error
	return res.ManifestId, err
}

func (w *Wdb) DelManifest(id string) error {
	return w.Db.Where("manifest_id = ?", id).Delete(&schema.Manifest{}).Error
}

func (w *Wdb) InsertApiKey(ak schema.AutoApiKey) error {
	return w.Db.Create(&ak).Error
}

func (w *Wdb) GetApiKeyDetail(key string) (schema.AutoApiKey, error) {
	res := schema.AutoApiKey{}
	err := w.Db.Model(&schema.AutoApiKey{}).Where("api_key = ?", key).First(&res).Error
	return res, err
}

func (w *Wdb) GetApiKeyDetailByAddress(addr string) (res schema.AutoApiKey, err error) {
	err = w.Db.Model(&schema.AutoApiKey{}).Where("address = ?", addr).First(&res).Error
	return
}

func (w *Wdb) ExistApikey(addr string) (bool, schema.AutoApiKey) {
	apikey, err := w.GetApiKeyDetailByAddress(addr)
	return err == nil, apikey
}

func (w *Wdb) UpdateApikeyTokenBal(addr string, newTokBal datatypes.JSONMap) error {
	return w.Db.Model(&schema.AutoApiKey{}).Where("address = ?", addr).Update("token_balance", newTokBal).Error
}

func (w *Wdb) GetApiKeyDepositRecords(addr string, cursorId int64, num int) ([]schema.ReceiptEverTx, error) {
	if cursorId <= 0 {
		cursorId = math.MaxInt64
	}
	records := make([]schema.ReceiptEverTx, 0, num)
	err := w.Db.Model(&schema.ReceiptEverTx{}).Where("raw_id < ? and `from` = ? and JSON_VALID(`data`) = 1 and JSON_CONTAINS(`data`, JSON_OBJECT('action', 'apikeyPayment')) = 1", cursorId, addr).Order("raw_id DESC").Limit(num).Find(&records).Error
	return records, err
}

func (w *Wdb) GetOrderRealTimeStatistic() ([]byte, error) {
	var results []schema.Result
	status := []string{"waiting", "pending", "success", "failed"}
	w.Db.Model(&schema.Order{}).Select("on_chain_status as status ,count(1) as totals,sum(size) as total_data_size").Group("on_chain_status").Find(&results)

	for _, s := range status {
		flag := true
		for i := range results {
			if s == results[i].Status {
				flag = false
			}
		}
		if flag {
			results = append(results, schema.Result{Status: s})
		}
	}
	return json.Marshal(results)
}

func (w *Wdb) GetOrderStatisticByDate(r schema.Range) ([]*schema.DailyStatistic, error) {
	var orderstatistics []schema.OrderStatistic
	start, _ := time.Parse("20060102", r.Start)
	end, _ := time.Parse("20060102", r.End)
	err := w.Db.Model(&schema.OrderStatistic{}).Where("date >= ? and date <= ?", start, end).Order("date").Find(&orderstatistics).Error
	if err != nil {
		return nil, err
	}
	res := make([]*schema.DailyStatistic, 0)
	for i := range orderstatistics {
		date := orderstatistics[i].Date.Format("20060102")
		res = append(res, &schema.DailyStatistic{
			Date: date,
			Result: schema.Result{
				Status:        schema.SuccOnChain,
				Totals:        orderstatistics[i].Totals,
				TotalDataSize: orderstatistics[i].TotalDataSize,
			},
		})
	}
	return res, nil
}

func (w *Wdb) GetDailyStatisticByDate(r schema.TimeRange) ([]schema.Result, error) {
	var results []schema.Result
	return results, w.Db.Model(&schema.Order{}).Select("count(1) as totals,sum(size) as total_data_size").Where("updated_at >= ? and updated_at < ? and on_chain_status = ?", r.Start, r.End, "success").Group("on_chain_status").Find(&results).Error
}

func (w *Wdb) WhetherExec(r schema.TimeRange) bool {
	var osc schema.OrderStatistic
	err2 := w.Db.Model(&schema.OrderStatistic{}).Where("date >= ? and date < ?", r.Start, r.End).First(&osc).Error
	return err2 != nil
}

func (w *Wdb) GetKafkaOrderInfos() ([]schema.KafkaOrderInfo, error) {
	results := make([]schema.KafkaOrderInfo, 0)
	err := w.Db.Model(&schema.Order{}).Where("expected_block > ? and payment_status = ? and kafka = ?", 1189450, schema.SuccPayment, false).Limit(50).Find(&results).Error
	return results, err
}

func (w *Wdb) KafkaDone(id uint) error {
	return w.Db.Model(&schema.Order{}).Where("id = ?", id).Update("kafka", true).Error
}
