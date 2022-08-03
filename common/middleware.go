package common

import (
	"encoding/base32"
	"errors"
	"fmt"
	"github.com/everFinance/goar/utils"
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"net/http"
	"regexp"
	"strings"
)

var (
	ERR_TOO_MANY_REQUESTS = errors.New("err_limit_exceeded")
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
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func SandboxMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		txId := getTxIdFromPath(c.Request.RequestURI)
		if txId != "" && c.GetHeader("x-amz-cf-id") == "" {
			currentSandbox := getRequestSandbox(c.Request)
			expectedSandbox := expectedTxSandbox(txId)
			if currentSandbox != expectedSandbox {
				c.Redirect(302, fmt.Sprintf("https://%s.%s%s", expectedSandbox, c.Request.Host, c.Request.RequestURI))
				return
			}
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

func getRequestSandbox(req *http.Request) string {
	hostStr := strings.Split(req.Host, ".")
	if len(hostStr) > 0 {
		return strings.ToLower(hostStr[0])
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
