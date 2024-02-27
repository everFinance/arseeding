package arseeding

import (
	"context"
	"encoding/json"
	"github.com/everFinance/arseeding/argraphql"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func Test_getRawById(t *testing.T) {
	data, contentType, err := getRawById("U1FqvR_xTuL2qxrJDw20oIghpGt1eTumJ9ZfCczc5_M")

	if err != nil {
		t.Error(err)
	}

	t.Log(err)
	t.Log(contentType)
	t.Log(string(data))

	mani := schema.ManifestData{}
	err = json.Unmarshal(data, &mani)
	t.Log(err)
}

func TestNewS3Store(t *testing.T) {
	err := syncManifestData("yy8F4i6jKVKtQuOw2q8RwxjQyUwJ4QtGRkdUXd8jbEw", nil)
	// err := syncManifestData("WUg9McRBT1i_F6utVYjFYS_pP6dzKcrC1FRccjH6inE", nil)
	t.Log(err)
}

func Test_getRawById1(t *testing.T) {
	data, contentType, err := getRawById("AjV6oRKHh5PPI8Ehu9hIyWEz3oFAm5K0I0UYkxjwLdE")
	assert.NoError(t, err)
	t.Log(contentType)
	bundle, err := utils.DecodeBundle(data)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(bundle.Items))

}

func Test_getNestBundle(t *testing.T) {
	itemIds := []string{"_5nCBFfMpHbpxRw6_hpoJyu8bV62S6flhUGrkdHbV8o"}
	items, err := getNestBundle("AjV6oRKHh5PPI8Ehu9hIyWEz3oFAm5K0I0UYkxjwLdE", itemIds)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(items))
}

func TestNewCache(t *testing.T) {
	nestBundle := "AjV6oRKHh5PPI8Ehu9hIyWEz3oFAm5K0I0UYkxjwLdE"
	gq := argraphql.NewARGraphQL("https://arweave.net/graphql", http.Client{})
	res, err := gq.QueryTransaction(context.Background(), nestBundle)
	assert.NoError(t, err)
	t.Log(res.GetTransaction().BundledIn.Id)
	t.Log(res.Transaction.Data.Size)
	t.Log(res.Transaction.Id)
}
