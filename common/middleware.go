package common

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"net/http"
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
