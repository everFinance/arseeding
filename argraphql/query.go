package argraphql

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"github.com/everFinance/go-everpay/common"
	"net/http"
)

var logger = common.NewLog("arseeding")

type ARGraphQL struct {
	Client graphql.Client
}

func NewARGraphQL(endpoint string, httpClient http.Client) *ARGraphQL {
	return &ARGraphQL{
		Client: graphql.NewClient(endpoint, &httpClient),
	}
}

func (g *ARGraphQL) QueryTransaction(ctx context.Context, id string) (res *GetTransactionResponse, err error) {

	txResp, err := GetTransaction(ctx, g.Client, id)

	if err != nil {
		logger.Error("ARGraphQL get transaction error", "err", err)
	}

	return txResp, err
}

func (g *ARGraphQL) BatchGetItemsBundleIn(ctx context.Context, ids []string, first int, after string) (res *BatchGetItemsBundleInResponse, err error) {

	batchResp, err := BatchGetItemsBundleIn(ctx, g.Client, ids, first, after)

	if err != nil {
		logger.Error("ARGraphQL batch get items bundle in error", "err", err)
	}

	return batchResp, err
}
