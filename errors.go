package arseeding

import (
	"errors"
)

var (
	ErrNotExist = errors.New("not_exist_record")
	ErrNotFound = errors.New("not_found")

	ErrExistTx    = errors.New("tx_exist")
	ErrExistJob   = errors.New("job_exist")
	ErrJobClosed  = errors.New("job_closed")
	ErrFetchData  = errors.New("fetch_tx_data_from_peers")
	ErrFetchArFee = errors.New("fetch_ar_fee_from_peers")

	ErrFullyLoaded = errors.New("fully_loaded")
	ErrDataTooBig  = errors.New("tx_data_too_big")
	ErrNullArId    = errors.New("null_ar_id")
	ErrNullData    = errors.New("null_data")
)
