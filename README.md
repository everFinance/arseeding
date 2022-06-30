# Arseeding
Arseeding is a lightweight arweave data seed node. It is mainly used to synchronize, cache and broadcast transaction && data.

Important: arseeding is compatible with all http api interfaces of arweave node and also provide bundle transaction api   
Related articles: [arseeding server design](https://medium.com/everfinance/arseeding-server-design-4e684176555a)

## Introduction
- `cmd` is the service starter
- `sdk` is used to call the arseeding api
- `bundle` is used to handle bundle items and get items info
- `cache` is used to cache important arweave info and reply to request quickly
- `api` register api and compatible with the implementation logic of all api's of arweave node and provide bundle service
- `jobs` is the implementation of timed jobs. This includes timed jobs that update some important info and roll up bundle Tx to arweave.
- `task` concurrent processing syncTasks,broadcastTasks and broadcastMetaTasks.
- `submit` includes the verification of transactions and chunks, and stores them in the bolt db.
- `wdb` store bundle transaction and its status
- `store` bolt db wrap.

## Run
```
PORT=':8080' KEY_PATH='yourKeyfilePath' MYSQL='mysq dsn' PAY='everpay api url' go run cmd/main.go
```

## Development
### Build & Run

```
make all
PORT=':8080' KEY_PATH='yourKeyfilePath' MYSQL='mysq dsn' PAY='everpay api url' ./build/arseeding
```

### Docker build
```
GOOS=linux GOARCH=amd64 go build -o ./cmd/arseeding ./cmd
docker build .
```

## API
arseeding is compatible with all http api interfaces of arweave node and also provide bundle api:
```
    {
        v1.POST("tx", s.submitTx)
		v1.POST("chunk", s.submitChunk)
		v1.GET("tx/:arid/offset", s.getTxOffset)
		v1.GET("/tx/:arid", s.getTx)
		v1.GET("chunk/:offset", s.getChunk)
		v1.GET("tx/:arid/:field", s.getTxField)
		v1.GET("/info", s.getInfo)
		v1.GET("/tx_anchor", s.getAnchor)
		v1.GET("/price/:size", s.getTxPrice)
		v1.GET("/peers", s.getPeers)
		// proxy
		v2 := r.Group("/")
		{
			v2.Use(proxyArweaveGateway)
			v2.GET("/tx/:arid/status")
			v2.GET("/price/:size/:target")
			v2.GET("/block/hash/:hash")
			v2.GET("/block/height/:height")
			v2.GET("/current_block")
			v2.GET("/wallet/:address/balance")
			v2.GET("/wallet/:address/last_tx")
			v2.POST("/arql")
			v2.POST("/graphql")
			v2.GET("/tx/pending")
			v2.GET("/unconfirmed_tx/:arId")
		}

		// broadcast && sync tasks
		v1.POST("/task/:taskType/:arid", s.postTask)
		v1.POST("/task/kill/:taskType/:arid", s.killTask)
		v1.GET("/task/:taskType/:arid", s.getTask)
		v1.GET("/task/cache", s.getCacheTasks)

		// ANS-104 bundle Data api
		v1.GET("/bundle/bundler", s.getBundler)
		v1.POST("/bundle/tx/:currency", s.submitItem)
		v1.GET("/bundle/tx/:itemId", s.getItemMeta) // get item meta, without data
		v1.GET("/bundle/itemIds/:arId", s.getItemIdsByArId)
		v1.GET("/bundle/fees", s.bundleFees)
		v1.GET("/bundle/fee/:size/:currency", s.bundleFee)
		v1.GET("bundle/orders/:signer", s.getOrders)
		v1.GET("/:id", s.getDataByGW) // get arTx data or bundleItem data
	}
```
note:    
when use `submitTx` and `submitChunk`, arseeding cache the tx and data and also submits it to the arweave gateway.   

sync and broadcast api:
```
	v1.POST("/task/:taskType/:arid", s.postTask)
	v1.POST("/task/kill/:taskType/:arid", s.killTask)
	v1.GET("/task/:taskType/:arid", s.getTask)
	v1.GET("/task/cache", s.getCacheTasks)
```
`killTask` This interface can be used to stop a broadcast task when enough nodes have been broadcast via `getTask`   
`taskType` is 'sync' or 'broadcast' or 'broadcast_meta'

`getCacheTasks` return all pending tasks

### bundle api describe:
```
    v1.GET("/bundle/bundler", s.getBundler)
	v1.POST("/bundle/tx/:currency", s.submitItem)
	v1.GET("/bundle/tx/:itemId", s.getItemMeta)
	v1.GET("/bundle/itemIds/:arId", s.getItemIdsByArId)
	v1.GET("/bundle/fees", s.bundleFees)
	v1.GET("/bundle/fee/:size/:currency", s.bundleFee)
	v1.GET("bundle/orders/:signer", s.getOrders)
	v1.GET("/:id", s.getDataByGW)
```
`getBundler` return a bundle service provider address
```
GET /bundle/bundler

resp: "Fkj5J8CDLC9Jif4CzgtbiXJBnwXLSrp5AaIllleH_yY"
```

`submitItem` submit a bundle item([goar](https://github.com/everFinance/goar/blob/bundle/bundleItem.go) is a useful tool to assemble a bundle item)
```
POST /bundle/tx/:currency
```
```
parameter:
```
| name   |type| value        |optional|
|--------|---|--------------|---|
|currency|string| AR |true|
note: 
1. if config charge no fee ,don't need currency, 
2. request header must be set "content-type"="application/octet-stream"
3. request body can be created use goar // see example/bundle-item/bundle_test.go
```
resp:
{
                ItemId:             "qe1231212441",
		Bundler:            "Fkj5J8CDLC9Jif4CzgtbiXJBnwXLSrp5AaIllleH_yY",
		Currency:           "AR",
		Decimals:           12,
		Fee:                "113123",
		PaymentExpiredTime: 122132421,
		ExpectedBlock:      3144212,
}
```


`bundleFees` return fees you need to pay for submit your bundle item to arweave and store it forever
```
GET /bundle/fees

resp:
{
    "AR": {
        "currency": "AR",
        "decimals": 12,
        "base": "808920",
        "perChunk": "66060288"
    },
    "DAI": {
        "currency": "DAI",
        "decimals": 18,
        "base": "7174878265432",
        "perChunk": "585934980689532"
    },
    "ETH": {
        "currency": "ETH",
        "decimals": 18,
        "base": "6585336377",
        "perChunk": "537790161708"
    },
    "USDC": {
        "currency": "USDC",
        "decimals": 6,
        "base": "7",
        "perChunk": "586"
    },
}
```
`getItemMeta` return item meta, without data
```
GET /bundle/tx/:itemId
```
| name   |type| value | optional |
|--------|---|----------|-------|
| itemId |string| IlYC5sG61mhTOlG2Ued5LWxN4nuhyZh3ror0MBbPKy4 | false |
```
resp:
{
    "signatureType": 3,
    "signature": "DC469T6Fz3ByFCtEjnP9AdsLSDFqINxvbIqFw1qwk0ApHtpmytRWFHZeY2gBN9nXopzY7Sbi9u5U6UcpPrwPlxs",
    "owner": "BCLR8qIeP8-kDAO6AifvSSzyCQJBwAtPYErCaX1LegK7GwXmyMvhzCmt1x6vLw4xixiOrI34ObhU2e1RGW5YNXo",
    "target": "",
    "anchor": "",
    "tags": [],
    "data": "",
    "id": "IlYC5sG61mhTOlG2Ued5LWxN4nuhyZh3ror0MBbPKy4"
}
```
`getOrders` return `signer address` all order
```
GET /bundle/orders/:signer
```
| name   |type| value | optional |
|--------|---|----------|-------|
| signer | string | Ii5wAMlLNz13n26nYY45mcZErwZLjICmYd46GZvn4ck | false |

```
resp:
[
    {
        "ID": 33,
        "CreatedAt": "2022-06-24T03:29:54.174Z",
        "UpdatedAt": "2022-06-24T04:30:09.193Z",
        "DeletedAt": null,
        "ItemId": "5rEb7c6OjMQIYjl6P7AJIb4bB9CLMBSxhZ9N7BVbRCk",
        "Signer": "Ii5wAMlLNz13n26nYY45mcZErwZLjICmYd46GZvn4ck",
        "SignType": 1,
        "Size": 1095,
        "Currency": "USDT",
        "Decimals": 6,
        "Fee": "701",
        "PaymentExpiredTime": 1656044994,
        "ExpectedBlock": 960751,
        "PaymentStatus": "expired",
        "PaymentId": "",
        "OnChainStatus": "failed"
    },
    {
        "ID": 32,
        "CreatedAt": "2022-06-23T09:56:45.402Z",
        "UpdatedAt": "2022-06-23T10:57:08.495Z",
        "DeletedAt": null,
        "ItemId": "BtAknheCYxMy_SoUbi8Mu1DuvWx9UDJzxvDSniu7Mf8",
        "Signer": "Ii5wAMlLNz13n26nYY45mcZErwZLjICmYd46GZvn4ck",
        "SignType": 1,
        "Size": 1095,
        "Currency": "USDT",
        "Decimals": 6,
        "Fee": "676",
        "PaymentExpiredTime": 1655981805,
        "ExpectedBlock": 960280,
        "PaymentStatus": "expired",
        "PaymentId": "",
        "OnChainStatus": "failed"
    },
    {
        "ID": 31,
        "CreatedAt": "2022-06-23T09:56:06.875Z",
        "UpdatedAt": "2022-06-23T10:56:08.499Z",
        "DeletedAt": null,
        "ItemId": "uNFlMoUE3yBbdM_EULhUftg8PY2umy3xSMHbnoguRJo",
        "Signer": "Ii5wAMlLNz13n26nYY45mcZErwZLjICmYd46GZvn4ck",
        "SignType": 1,
        "Size": 1095,
        "Currency": "USDT",
        "Decimals": 6,
        "Fee": "676",
        "PaymentExpiredTime": 1655981766,
        "ExpectedBlock": 960279,
        "PaymentStatus": "expired",
        "PaymentId": "",
        "OnChainStatus": "failed"
    },
]
```

`getDataByGW` get arTx data or bundleItem data
```
GET /:id
```
| name |type| value | optional |
|------|---|----------|-------|
| id   | string | IlYC5sG61mhTOlG2Ued5LWxN4nuhyZh3ror0MBbPKy4 | false |
```
resp:
byte data
```
## Usage
### compatible arweave sdk
[arweave-js](https://github.com/ArweaveTeam/arweave-js) sdk
```
import Arweave from 'arweave';

const arweave = Arweave.init({
    host: '127.0.0.1', // arseeding service url
    port: 8080,
    protocol: 'http'
});
```
[goar](https://github.com/everFinance/goar) sdk
```
arNode := "http://127.0.0.1:8080" // arseeding service url
arClient := goar.NewClient(arNode) 
```
### Different
Arseeding is a light node, so it does not store all the data in the arweave network, so when requesting tx data, it is likely that the data will not be available, even if the data already exists in the arweave network.    
In the case we use the `/task/sync/:arid` api of arseeding to synchronize the tx to the service.   
e.g:
1. User want to get a tx
``` 
 arId := "yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8"

 // connect arseeding server by goar sdk
 arClient := goar.NewClient("http://127.0.0.1:8080") 
 
 data, err := arClient.GetTransactionData(arId)
 // err: not found
```
By default arseeding does not contain this data, so will return 'not found' error msg.   

2. So we need to use arseeding `sync` api
```
curl --request POST 'http://127.0.0.1:8080/task/sync/yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8'
```
3. Use `getTask` api to watcher the job status
```
 curl 'http://127.0.0.1:8080/task/sync/yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8'
```
resp:
```
{
    "arId": "yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8",
    "tktype": "sync",
    "countSuccessed": 1,
    "countFailed": 0,
    "totalNodes": 945,
    "close": false
}
```
`countSuccessed` 1 means sync success

4. This time re-run step 1 and it will work.

## Broadcast usage
If you want your tx or data to be broadcast to all nodes, the bardcast function can help you do that.   
e.g:
1. Register for tx that require broadcasting
```
curl --request POST 'http://127.0.0.1:8080/task/broadcast/yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8'
```
2. Use `getTask` api to watcher the job status
```
curl GET 'http://127.0.0.1:8080/task/broadcast/yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8'
```
resp:
```
{
    "arid": "yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8",
    "tktype": "broadcast",
    "countSuccessed": 220,
    "countFailed": 9,
    "totalNodes": 939,
    "close": false
}
```
`countSuccessed` number of nodes successfully broadcast   
`countFailed` number of nodes failed to broadcast   
`totalNodes` total number of nodes   
`close` Is the broadcast task closed   

3. If the goal is to successfully broadcast to 200 nodes, then this broadcast task can be closed
```
curl --request POST 'http://127.0.0.1:8080/task/kill/broadcast/yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8'
```

### Bundle Usage
Arseeding implement ANS-104 bundle Data API, [goar](https://github.com/everFinance/goar/tree/bundle) is a useful tool to assemble bundle item and interact with these APIs

`e.g:`
```
    // connect arseeding server by goar sdk
    signer, err := goar.NewSignerFromPath("test-keyfile.json")
	arseedUrl := "http://127.0.0.1:8080"
	itemSdk, err := goar.NewItemSdk(signer, arseedUrl)
	
    // now just use itemSdk to create a bundle item
    item, err := itemSdk.CreateAndSignItem(dataFiled, target, anchor, tags)
    
    // submit bundle item to arseeding
    resp, err := itemSdk.SubmitItem(item, "AR") // "AR" can be replace with any Token that the bundlr support
    // you will get info about how much you should pay, bundlr address, bundle id that can be used query your item from resp struct
```
resp:
```
{
    ItemId             string
    Bundler            string
    Currency           string
    Fee                string
    PaymentExpiredTime int64
    ExpectedBlock      int64
}
```



## Example
[everPay rollup txs sync](https://github.com/everFinance/arseeding/tree/main/example/everpay-sync): get all everpay rollup txIds from the arweave node, and then post to the arseeding service using the `sync` interface.

[broadcast arweave pending pool txs](https://github.com/everFinance/arseeding/tree/main/example/arweave-pool-broadcast): get pending pool txIds, sync to arseeding and broadcast to all nodes.
## Notes
1. The tx and data must exist in the arseeding service before using broadcast. In other words, if there is no tx and data in arseeding, use sync to synchronize the data before using broadcast.
2. The broadcast job needs to commit tx and data to each node, so each broadcast job takes a long time to execute. So the  user checks the number of successful broadcast nodes by `getJob`, and then actively stops the broadcast job by `killJob`. 

## Licenses
Users are requested to comply with their own license   
[Apache License](https://github.com/everFinance/arseeding/blob/main/LICENSE)