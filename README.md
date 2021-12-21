# Arseeding
Arseeding is a lightweight arweave data seed node. It is mainly used to synchronize, cache and broadcast transaction && data.

Important: arseeding is compatible with all http api interfaces of arweave node.

## Introduction
- `cmd` is the service starter
- `api-job` implements the service logic of the api interface for sync and broadcast.
- `api` register api and compatible with the implementation logic of all api's of arweave node.
- `job-manager` is the sync and broadcast job manager.
- `jobs` is the implementation of timed tasks. This includes timed tasks that handle syncJobs and broadcastJobs.
- `service` logic related to the submitTx and submitChunk interfaces.
- `store` bolt db wrap.

## Run
```
PORT=':8080' go run cmd/main.go
```

## Development
### Build & Run

```
make all
PORT=':8080' ./build/arseeding
```

### Docker

TODO

## API
arseeding is compatible with all http api interfaces of arweave node:
```
    v1.POST("tx", s.submitTx)
    v1.POST("chunk", s.submitChunk)
    v1.GET("tx/:arid/offset", s.getTxOffset)
    v1.GET("/tx/:arid", s.getTx)
    v1.GET("chunk/:offset", s.getChunk)
    v1.GET("tx/:arid/:field", s.getTxField)
    // proxy
    v2 := r.Group("/")
    {
        v2.Use(proxyArweaveGateway)
        v2.GET("/info")
        v2.GET("/tx/:arid/status")
        v2.GET("/:arid")
        v2.GET("/price/:size")
        v2.GET("/price/:size/:target")
        v2.GET("/block/hash/:hash")
        v2.GET("/block/height/:height")
        v2.GET("/current_block")
        v2.GET("/wallet/:address/balance")
        v2.GET("/wallet/:address/last_tx")
        v2.GET("/peers")
        v2.GET("/tx_anchor")
        v2.POST("/arql")
        v2.POST("/graphql")
        v2.GET("/tx/pending")
        v2.GET("/unconfirmed_tx/:arId")
    }
```
note:    
when use `submitTx` and `submitChunk`, arseeding caches the tx and data and also submits it to the arweave gateway.   

sync and broadcast api:
```
    v1.POST("/job/broadcast/:arid", s.broadcast)
    v1.POST("/job/sync/:arid", s.sync)
    v1.POST("/job/kill/:arid/:jobType", s.killJob)
    v1.GET("/job/:arid/:jobType", s.getJob)
    v1.GET("/cache/jobs", s.getCacheJobs)
```
`killJob` This interface can be used to stop a broadcast job when enough nodes have been broadcast via `getJob`   
`jobType` is 'sync' or 'broadcast'   
`getCacheJobs` return all pending jobs

## Usage
### compatible arweave sdk
arweave-js sdk
```
import Arweave from 'arweave';

const arweave = Arweave.init({
    host: '127.0.0.1', // arseeding service url
    port: 8080,
    protocol: 'http'
});
```
goar sdk
```
arNode := "http://127.0.0.1:8080" // arseeding service url
arClient := goar.NewClient(arNode) 
```
### Different
Arseeding is a light node, so it does not store all the data in the arweave network, so when requesting tx or tx data, it is likely that the data will not be available, even if the data already exists in the arweave network.    
In the case we use the `sync` api of arseeding to synchronize the tx to the service.   
e.g:
1. User want to get a tx
``` 
 arId := "yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8"

 // connect arseeding server by goar sdk
 arClient := goar.NewClient("http://127.0.0.1:8080") 
 
 tx, err := arClient.GetTransactionById(arId)
 // or 
 data, err := arClient.GetTransactionData(arId)
 // err: not found
```
By default arseeding does not contain this data, so will return 'not found' error msg.   

2. So we need to use arseeding `sync` api
```
curl --request POST 'http://127.0.0.1:8080/job/sync/yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8'
```
3. Use `getJob` api to watcher the job status
```
 curl 'http://127.0.0.1:8080/job/yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8/sync'
```
resp:
```
{
    "arId": "yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8",
    "jobType": "sync",
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
curl --request POST 'http://127.0.0.1:8080/job/broadcast/yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8'
```
2. Use `getJob` api to watcher the job status
```
curl GET 'http://127.0.0.1:8080/job/yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8/broadcast'
```
resp:
```
{
    "arid": "yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8",
    "jobType": "broadcast",
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
curl --request POST 'http://127.0.0.1:8080/job/kill/yK_x7-bKBOe1GK3sEHWIQ4QZRibn504pzYOFa8iO2S8/broadcast'
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