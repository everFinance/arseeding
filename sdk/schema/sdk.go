package schema

import "github.com/everFinance/goar/types"

type OptionItem struct {
	Target       string
	Anchor       string
	NeedSequence bool
	Tags         []types.Tag
}
