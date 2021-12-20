# ARseeding
ARseeding is a lightweight arweave data seed node. It is mainly used to synchronize, cache and broadcast transaction && data.

Important: arseeding is compatible with all http api interfaces of arweave node.

# code introduction
- `cmd` is the service starter
- `api-job` implements the service logic of the api interface for sync and broadcast.
- `api` register api and compatible with the implementation logic of all api's of arweave node.
- `job-manager` is the sync and broadcast job manager.
- `jobs` is the implementation of timed tasks. This includes timed tasks that handle syncJobs and broadcastJobs.
- `service` logic related to the submitTx and submitChunk interfaces.
- `store` bolt db wrap.

# start
```
go run cmd/main.go
```
# API
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











