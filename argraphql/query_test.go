package argraphql

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/assert"
	"net/http"
	"reflect"
	"testing"
)

func TestARGraphQL_QueryTransaction(t *testing.T) {
	type fields struct {
		Client graphql.Client
	}
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantRes *GetTransactionResponse
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				Client: graphql.NewClient("https://arweave.net/graphql", &http.Client{}),
			},
			// oLQYpPmfv35ZGn5Vv0_H2GKLoyY5bnFAChngzjLiCEs
			args: args{
				ctx: context.Background(),
				id:  "oLQYpPmfv35ZGn5Vv0_H2GKLoyY5bnFAChngzjLiCEs",
			},
			/**
			{
				"transaction": {
					"id": "oLQYpPmfv35ZGn5Vv0_H2GKLoyY5bnFAChngzjLiCEs",
					"anchor": "MDh5WGFnK1lieTdKekl4K25LaFZkRjRjQ0s2YU9BdmM",
					"signature": "hiHfDFJ0e6NFTooQwUg5hpKqtjBQAO50ApSILtQw_arT1PZ0oT8IapqxPg_iKUmAjyfS6rFHusbZkckahfP0_2sodoEEwv22XFcoRVJR3IlEgDBPAkrx2p7jZXN25EalqdUby6CUBusU54bUU4COhu3NTQQ5E7QyWo-hadCQm1mPQXxWV0sceS1qnhhsnDLO0vb9AOXt5P6zd5GCgbFDgxMZXGFA5-RfnyfIQwEe2kttwfxuLgfPqJKV7hUFB55rDDZXfw4Ps-PTUKJMSrwpRm95aWobFe9uKmf-LGFBZ-CnvxW0Jr6qf8QyAg7LzmboxIKClXV1QBbQzjlacGP_pZQFQLMna6k8VFZsja_L4jkftubgvdxQJygzWFVuXD0RiY0IwOVMDyrQePgRJXFi2kKoDGoF71ulbkL7dMVejQ6wb-7xc7wZp0cJU-Gk50mY70ASO-3tcCfs6_F1BRv7Mpt0P8Jq2GZKNRA7NVQVt6hImd-S8hh2_uVa-Q3R1YExnp5duP4LZGgiryXg_e0M1B895aPXdAktSbvJMButnWy_rH6TBiMA14q-Hqa8TDMiV_PWGJSM1cKj2qKZlqJj_UO_BFT09uNzQzMZKG3iwZI1qXqTddY_NjjsKkrezCAzCQG-A517tGhSA8KG7v6WjMD2gYVJxNkF3tQaPQ_joIc",
					"recipient": "",
					"owner": {
						"address": "wQH9ojMXplWYiQvkSGVjIdWclk3XDM59poFoaOXKR8E",
						"key": "l_V57UmpROcyH6bYlIjeflxe6YRLDkeC0bhF4yUX7140wwbTGXnF-WJQNoG3XHTvMdeSf4eCMF9G5V8uxnXWYs2PvGjR_5l_Ox-IWR2JIA8118oBZ1FQh9gkkSShoDQK4dJ1EmIQVJZFAmyx370COnM5kCU1yFHMFM3XfBPG9PQDgewpXL2haFzwDEq2jNff_q-cmPr1tmZY0gTQW_l-3aeruteZg9VMg-JK5sRann4L53dvP4DJ2LLM585E_et4cSTf_3gCkOwoykp3UM61UaeNBi31xeDfH2sd73id1sP-vG2NO1BOqL2PGWudC3z9gbU9CzlF8juf927s9MszlJf6IhfjwgVcmu3BKBmJ9D9yhMrXlLi_v5TT1R7e-m7-6JSDAT9oNs3KRQ-B7gZ-kWhWuZZdgb7KW5N2q5EMAXu8a8WVipicGMXG5y6IIt8aPtHM0MzUOiCN-956sXTkF546pysMtQHYv5A1ctUgjwg5L5-rKLi_nrlH5t6aEmP4TJwDA122kYIa4qYz0Vfle7DP94et5lwPVP2ofzgyVVzpCmxLI53uqbHy5pPpzLVH_JB4bVgCYjK3ddtlNernufUV5kRhAVS48fPZKQ2Y-gpR4908i6A5EnGxYbqkY8Q2-fgqX2hc17xFXLdzsMiwPxO1z7rg14BGPEH9xY3aMY0"
					},
					"fee": {
						"winston": "0",
						"ar": "0.000000000000"
					},
					"quantity": {
						"winston": "0",
						"ar": "0.000000000000"
					},
					"data": {
						"type": "",
						"size": "818"
					},
					"tags": [{
						"name": "Content-Type",
						"value": "application/javascript; charset=utf-8"
					}],
					"block": {
						"id": "-pOVy3h66x40s0ITVN55FVUAzQU3b56PXCr9DjQasnKV_I855cheOPU4DWHPBzne",
						"timestamp": 1689883479,
						"height": 1224045,
						"previous": "IeBQvT4sX5iLOrDLXmd3AaWIwI5p9sb9ktN1RUJUiJeFmnCM6ztn_EBMy2maSAVp"
					},
					"bundledIn": {
						"id": "D0i_pTTxNo_e1csmGSI-rJvKz6xi61ksQVjLEig568E"
					}
				}
			}

			full json to strut
			*/
			wantRes: &GetTransactionResponse{
				Transaction: GetTransactionTransaction{

					Id:        "D0i_pTTxNo_e1csmGSI-rJvKz6xi61ksQVjLEig568E",
					Anchor:    "MDh5WGFnK1lieTdKekl4K25LaFZkRjRjQ0s2YU9BdmM",
					Signature: "hiHfDFJ0e6NFTooQwUg5hpKqtjBQAO50ApSILtQw_arT1PZ0oT8IapqxPg_iKUmAjyfS6rFHusbZkckahfP0_2sodoEEwv22XFcoRVJR3IlEgDBPAkrx2p7jZXN25EalqdUby6CUBusU54bUU4COhu3NTQQ5E7QyWo-hadCQm1mPQXxWV0sceS1qnhhsnDLO0vb9AOXt5P6zd5GCgbFDgxMZXGFA5-RfnyfIQwEe2kttwfxuLgfPqJKV7hUFB55rDDZXfw4Ps-PTUKJMSrwpRm95aWobFe9uKmf-LGFBZ-CnvxW0Jr6qf8QyAg7LzmboxIKClXV1QBbQzjlacGP_pZQFQLMna6k8VFZsja_L4jkftubgvdxQJygzWFVuXD0RiY0IwOVMDyrQePgRJXFi2kKoDGoF71ulbkL7dMVejQ6wb-7xc7wZp0cJU-Gk50mY70ASO-3tcCfs6_F1BRv7Mpt0P8Jq2GZKNRA7NVQVt6hImd-S8hh2_uVa-Q3R1YExnp5duP4LZGgiryXg_e0M1B895aPXdAktSbvJMButnWy_rH6TBiMA14q-Hqa8TDMiV_PWGJSM1cKj2qKZlqJj_UO_BFT09uNzQzMZKG3iwZI1qXqTddY_NjjsKkrezCAzCQG-A517tGhSA8KG7v6WjMD2gYVJxNkF3tQaPQ_joIc",
					Recipient: "",
					Owner: GetTransactionTransactionOwner{
						Address: "wQH9ojMXplWYiQvkSGVjIdWclk3XDM59poFoaOXKR8E",
						Key:     "l_V57UmpROcyH6bYlIjeflxe6YRLDkeC0bhF4yUX7140wwbTGXnF-WJQNoG3XHTvMdeSf4eCMF9G5V8uxnXWYs2PvGjR_5l_Ox-IWR2JIA8118oBZ1FQh9gkkSShoDQK4dJ1EmIQVJZFAmyx370COnM5kCU1yFHMFM3XfBPG9PQDgewpXL2haFzwDEq2jNff_q-cmPr1tmZY0gTQW_l-3aeruteZg9VMg-JK5sRann4L53dvP4DJ2LLM585E_et4cSTf_3gCkOwoykp3UM61UaeNBi31xeDfH2sd73id1sP-vG2NO1BOqL2PGWudC3z9gbU9CzlF8juf927s9MszlJf6IhfjwgVcmu3BKBmJ9D9yhMrXlLi_v5TT1R7e-m7-6JSDAT9oNs3KRQ-B7gZ-kWhWuZZdgb7KW5N2q5EMAXu8a8WVipicGMXG5y6IIt8aPtHM0MzUOiCN-956sXTkF546pysMtQHYv5A1ctUgjwg5L5-rKLi_nrlH5t6aEmP4TJwDA122kYIa4qYz0Vfle7DP94et5lwPVP2ofzgyVVzpCmxLI53uqbHy5pPpzLVH_JB4bVgCYjK3ddtlNernufUV5kRhAVS48fPZKQ2Y-gpR4908i6A5EnGxYbqkY8Q2-fgqX2hc17xFXLdzsMiwPxO1z7rg14BGPEH9xY3aMY0",
					},
					Fee: GetTransactionTransactionFeeAmount{
						Winston: "0",
						Ar:      "0.000000000000",
					},
					Quantity: GetTransactionTransactionQuantityAmount{
						Winston: "0",
						Ar:      "0.000000000000",
					},
					Data: GetTransactionTransactionDataMetaData{
						Type: "",
						Size: "818",
					},
					Tags: []GetTransactionTransactionTagsTag{
						{
							Name:  "Content-Type",
							Value: "application/javascript; charset=utf-8",
						},
					},
					Block: GetTransactionTransactionBlock{
						Id:        "-pOVy3h66x40s0ITVN55FVUAzQU3b56PXCr9DjQasnKV_I855cheOPU4DWHPBzne",
						Timestamp: 1689883479,
						Height:    1224045,
						Previous:  "IeBQvT4sX5iLOrDLXmd3AaWIwI5p9sb9ktN1RUJUiJeFmnCM6ztn_EBMy2maSAVp",
					},
					BundledIn: GetTransactionTransactionBundledInBundle{
						Id: "D0i_pTTxNo_e1csmGSI-rJvKz6xi61ksQVjLEig568E",
					},
				},
			},

			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &ARGraphQL{
				Client: tt.fields.Client,
			}
			gotRes, err := g.QueryTransaction(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryTransaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("QueryTransaction() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func TestNewARGraphQL(t *testing.T) {
	type args struct {
		endpoint   string
		httpClient http.Client
	}
	tests := []struct {
		name string
		args args
		want *ARGraphQL
	}{
		{
			name: "test",
			args: args{
				endpoint:   "http://localhost:1984",
				httpClient: http.Client{},
			},
			want: &ARGraphQL{
				Client: graphql.NewClient("http://localhost:1984", &http.Client{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewARGraphQL(tt.args.endpoint, tt.args.httpClient); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewARGraphQL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryTransaction(t *testing.T) {

	gq := NewARGraphQL("https://arweave.net/graphql", http.Client{})

	res, err := gq.QueryTransaction(context.Background(), "oLQYpPmfv35ZGn5Vv0_H2GKLoyY5bnFAChngzjLiCEs")

	assert.NoError(t, err)

	assert.Equal(t, "oLQYpPmfv35ZGn5Vv0_H2GKLoyY5bnFAChngzjLiCEs", res.Transaction.Id)

	assert.Equal(t, "0.000000000000", res.Transaction.Fee.Ar)

	assert.LessOrEqualf(t, "application/javascript; charset=utf-8", res.Transaction.Tags[0].Value, "tag name should be less than or equal to application/javascript; charset=utf-8")

	assert.Equal(t, "D0i_pTTxNo_e1csmGSI-rJvKz6xi61ksQVjLEig568E", res.Transaction.BundledIn.Id)
}

func TestBatchGetItemsBundleIn(t *testing.T) {

	gq := NewARGraphQL("https://arweave.net/graphql", http.Client{})
	ids := []string{"arDRw5qt51v4pOV9TrQXKJM2iLK-c39dvs2K-7b3oDk", "O0N7iKmdv7Tmc0fnvJSKSeKuibvDrHbpMlb4K8pHwXg", "oLQYpPmfv35ZGn5Vv0_H2GKLoyY5bnFAChngzjLiCEs"}

	res, err := gq.BatchGetItemsBundleIn(context.Background(), ids, 100, "")

	assert.NoError(t, err)
	assert.False(t, res.Transactions.PageInfo.HasNextPage)
	assert.Equal(t, 3, len(res.Transactions.Edges))

	for _, edge := range res.Transactions.Edges {
		if edge.Node.Id == "arDRw5qt51v4pOV9TrQXKJM2iLK-c39dvs2K-7b3oDk" {
			assert.Equal(t, "FnzJJ_6TDcgapyvs_-8vL2ImIWwehvRp_aWdhwS57U0", edge.Node.BundledIn.Id)
		}

		if edge.Node.Id == "O0N7iKmdv7Tmc0fnvJSKSeKuibvDrHbpMlb4K8pHwXg" {
			assert.Equal(t, "FnzJJ_6TDcgapyvs_-8vL2ImIWwehvRp_aWdhwS57U0", edge.Node.BundledIn.Id)
		}

		if edge.Node.Id == "oLQYpPmfv35ZGn5Vv0_H2GKLoyY5bnFAChngzjLiCEs" {
			assert.Equal(t, "D0i_pTTxNo_e1csmGSI-rJvKz6xi61ksQVjLEig568E", edge.Node.BundledIn.Id)
		}
	}

	t.Log(res)

}
