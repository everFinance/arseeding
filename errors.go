package arseeding

import (
	"errors"
)

var (
	ErrNotExist = errors.New("not_exist_record")
	ErrNotFound = errors.New("not_found")

	ErrExistTx    = errors.New("tx_exist")
	ErrTaskClosed = errors.New("task_closed")
	ErrFetchData  = errors.New("fetch_tx_data_from_peers")
	ErrFetchArFee = errors.New("fetch_ar_fee_from_peers")

	ErrDataTooBig = errors.New("tx_data_too_big")
	ErrNullData   = errors.New("null_data")
)
