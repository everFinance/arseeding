package schema

type DomainRecord struct {
	ProcessId      string `json:"processId"`
	StartTimestamp int64  `json:"startTimestamp"`
	Type           string `json:"type"`
	EndTimestamp   int64  `json:"endTimestamp"`
	PurchasePrice  int64  `json:"purchasePrice"`
	UndernameLimit int64  `json:"undernameLimit"`
}

type DomainState struct {
	Controllers []string `json:"Controllers"`
	Records     map[string]struct {
		TransactionId string `json:"transactionId"`
		TtlSeconds    int64  `json:"ttlSeconds"`
	}
	Name           string           `json:"Name"`
	Denomination   int64            `json:"Denomination"`
	Logo           string           `json:"Logo"`
	Ticker         string           `json:"Ticker"`
	Owner          string           `json:"Owner"`
	Initialized    bool             `json:"Initialized"`
	TotalSupply    int64            `json:"TotalSupply"`
	Balances       map[string]int64 `json:"Balances"`
	SourceCodeTXID string           `json:"Source-Code-TX-ID"`
}
