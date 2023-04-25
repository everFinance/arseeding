package schema

var (
	// bucket
	ChunkBucket           = "chunk-bucket"              // key: chunkStartOffset, val: chunk
	TxDataEndOffSetBucket = "tx-data-end-offset-bucket" // key: dataRoot+dataSize; val: txDataEndOffSet
	TxMetaBucket          = "tx-meta-bucket"            // key: txId, val: arTx; not include data
	ConstantsBucket       = "constants-bucket"

	// tasks
	TaskIdPendingPoolBucket = "task-pending-pool-bucket" // key: taskId(taskType+"-"+arId), val: "0x01"
	TaskBucket              = "task-bucket"              // key: taskId(taskType+"-"+arId), val: task

	// bundle bucketName
	BundleItemBinary = "bundle-item-binary"
	BundleItemMeta   = "bundle-item-meta"

	// parse arTx data to bundle items
	BundleWaitParseArIdBucket = "bundle-wait-parse-arId-bucket" // key: arId, val: "0x01"
	BundleArIdToItemIdsBucket = "bundle-arId-to-itemIds-bucket" // key: arId, val: json.marshal(itemIds)

	//statistic
	StatisticBucket = "order-statistic-bucket"
)
