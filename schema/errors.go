package schema

import (
	"errors"
)

var (
	ErrNotExist = errors.New("not_exist_record")
	ErrNotFound = errors.New("not_found")
	ErrExist    = errors.New("s3_bucket_exist")

	ErrExistTx    = errors.New("tx_exist")
	ErrTaskClosed = errors.New("task_closed")
	ErrFetchData  = errors.New("fetch_tx_data_from_peers")

	ErrDataTooBig    = errors.New("tx_data_too_big")
	ErrNullData      = errors.New("null_data")
	ErrLocalNotExist = errors.New("not_exist_local") // need to get data from gateway
	ErrPageNotFound  = errors.New("page_not_found")  // e.g manifest data not contain index path
	ErrNotImplement  = errors.New("method not implement")
)
