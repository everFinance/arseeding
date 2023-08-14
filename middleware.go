package arseeding

import (
	"encoding/base32"
	"errors"
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar/utils"
	"github.com/everFinance/goarns"
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	ERR_TOO_MANY_REQUESTS = errors.New("err_limit_exceeded")
	MANIFEST_ID_NOT_FOUND = errors.New("err_manifest_id_not_found")
)

// LimiterMiddleware period: "S"<Second>,"M"<Minute>,"H"<Hour>,"D"<Day>; limit: limit frequency
func LimiterMiddleware(limit int, period string, ipRateWhitelist *map[string]struct{}) gin.HandlerFunc {
	rate, err := limiter.NewRateFromFormatted(fmt.Sprintf("%d-%s", limit, period))
	if err != nil {
		panic(err)
	}
	store := memory.NewStore()
	middleware := mgin.NewMiddleware(limiter.New(store, rate),
		mgin.WithLimitReachedHandler(func(c *gin.Context) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": ERR_TOO_MANY_REQUESTS.Error(),
			})
		}),
		mgin.WithKeyGetter(func(c *gin.Context) string {
			return c.Request.Header.Get("origin") + "," + c.ClientIP()
		}),
		mgin.WithExcludedKey(func(originAndIp string) bool { // origin + "," + ip
			if ipRateWhitelist == nil {
				return false
			}
			mmap := *ipRateWhitelist
			ss := strings.Split(originAndIp, ",")
			for _, s := range ss {
				if _, ok := mmap[s]; ok {
					return true
				}
			}
			return false
		}))

	return middleware
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Sort, sort, X-API-KEY, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, HEAD")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func ManifestMiddleware(s *Arseeding) gin.HandlerFunc {

	wdb := s.wdb
	store := s.store
	localCache := s.localCache

	return func(c *gin.Context) {
		prefixUri := getRequestSandbox(c.Request.Host)
		if len(prefixUri) > 0 && c.Request.Method == "GET" {
			// compatible url https://xxxxxxx.arseed.web3infra.dev/{{arId}}
			txId := getTxIdFromPath(c.Request.RequestURI)
			if txId != "" && prefixUri == expectedTxSandbox(txId) {
				protocol := "https"
				if c.Request.TLS == nil {
					protocol = "http"
				}
				// redirect url: https://arseed.web3infra.dev/{{arId}}
				rootHost := strings.SplitN(c.Request.Host, ".", 2)[1]
				redirectUrl := fmt.Sprintf("%s://%s/%s", protocol, rootHost, txId)
				c.Redirect(302, redirectUrl)

				c.Abort()
				return
			}

			mfId, err := wdb.GetManifestId(prefixUri)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": MANIFEST_ID_NOT_FOUND.Error(),
				})
				c.Abort()
				return
			}

			_, dataReader, mfData, err := getArTxOrItemData(mfId, store)
			defer func() {
				if dataReader != nil {
					dataReader.Close()
					os.Remove(dataReader.Name())
				}
			}()
			if err != nil {
				c.Abort()
				internalErrorResponse(c, err.Error())
				return
			}
			if dataReader != nil {
				mfData, err = io.ReadAll(dataReader)
				if err != nil {
					c.Abort()
					internalErrorResponse(c, err.Error())
					return
				}
			}
			tags, data, err := handleManifest(mfData, c.Request.URL.Path, store)
			if err != nil {
				c.Abort()
				internalErrorResponse(c, err.Error())
				return
			}
			c.Abort()
			c.Data(http.StatusOK, fmt.Sprintf("%s; charset=utf-8", getTagValue(tags, schema.ContentType)), data)
			return
		}

		//  Gateway  logic
		currentHost := c.Request.Host
		domain := getSubDomain(c.Request.Host)
		// default  true
		notApiHost := true
		apiHostList := []string{
			"seed-dev.everpay.io",
			"arseed.web3infra.dev",
			"web3infura.io",
			"arweave.world",
			"arweave.asia",
		}

		log.Debug("middleware", "currentHost", currentHost)
		for _, b := range apiHostList {

			if currentHost == b {
				// if current host in apiHostList, notApiHost is false
				notApiHost = false
				break
			}
		}

		// if domain is not empty and method is get and not in apiHostList
		if len(domain) > 0 && c.Request.Method == "GET" && notApiHost {

			txId := ""
			keyPrefix := "txId_"

			// get txId by localCache
			value, err := localCache.Cache.Get(keyPrefix + domain)
			if err != nil {
				log.Info("middleware get localCache ", "error:", err)
			}

			if value != nil {
				txId = string(value)
			}

			// if txId is empty, get txId by sdk
			if txId == "" {

				// todo config or variable
				dreUrl := "https://dre-3.warp.cc"
				arNSAddress := "bLAgYxAdX2Ry-nt6aH2ixgvJXbpsEYm28NgJgyqfs-U"
				timeout := 10 * time.Second

				a := goarns.NewArNS(dreUrl, arNSAddress, timeout)

				txId, err = a.QueryLatestRecord(domain)
				if err != nil {
					c.Abort()
					internalErrorResponse(c, err.Error())
					return
				}

				// set cache
				err = localCache.Cache.Set(keyPrefix+domain, []byte(txId))

				if err != nil {
					log.Error("middleware set localCache", "error", err)
				}
			}

			log.Debug(fmt.Sprintf("permaweb domian: %s txId: %s", domain, txId))
			decodeTags, dataReader, mfData, err := getArTxOrItemDataForManifest(txId, store, s)
			defer func() {
				if dataReader != nil {
					dataReader.Close()
					os.Remove(dataReader.Name())
				}
			}()

			// if err equal to schema.ErrLocalNotExist, return 200 and "syncing data please wait some minutes and fresh the page"
			if err == schema.ErrLocalNotExist {
				c.Abort()
				c.JSON(http.StatusOK, gin.H{
					"msg": "syncing data please wait some minutes and fresh the page",
				})
				return
			}

			if err != nil {
				c.Abort()
				internalErrorResponse(c, err.Error())
				return
			}
			if dataReader != nil {
				mfData, err = io.ReadAll(dataReader)
				if err != nil {
					c.Abort()
					internalErrorResponse(c, err.Error())
					return
				}
			}

			// if content type is text/html, return mfData
			if getTagValue(decodeTags, schema.ContentType) == "text/html" {
				c.Abort()
				c.Data(http.StatusOK, fmt.Sprintf("%s; charset=utf-8", getTagValue(decodeTags, schema.ContentType)), mfData)
				return
			}

			tags, data, err := handleManifest(mfData, c.Request.URL.Path, store)
			if err != nil {
				c.Abort()
				internalErrorResponse(c, err.Error())
				return
			}
			c.Abort()
			c.Data(http.StatusOK, fmt.Sprintf("%s; charset=utf-8", getTagValue(tags, schema.ContentType)), data)
			return

		}
		c.Next()
	}
}

func getTxIdFromPath(path string) string {
	reg1 := regexp.MustCompile(`^\/?([a-zA-Z\d-_]{43})`)
	matchs := reg1.FindAllStringSubmatch(path, -1)
	if len(matchs) > 0 && len(matchs[0]) > 1 {
		return matchs[0][1]
	}
	return ""
}

func getRequestSandbox(host string) string {
	prefix := strings.Split(host, ".")[0]
	if len(prefix) > 40 { // todo 40
		return prefix
	}
	return ""
}

func expectedTxSandbox(txId string) string {
	txId = replaceId(txId)
	by32, _ := utils.Base64Decode(txId)
	res := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(by32)
	return strings.ToLower(res)
}

func replaceId(txId string) string {
	byteArr := make([]byte, 0)
	for i := 0; i < len(txId); i++ {
		if txId[i] != '-' && txId[i] != '_' {
			byteArr = append(byteArr, txId[i])
		}
	}
	return string(byteArr)
}

func getSubDomain(domain string) string {
	return strings.Split(domain, ".")[0]
}
