package middleware

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	// Do not use wildcard with credentials (browsers reject it and it risks token leakage).
	// AllowOriginFunc reflects any origin when no explicit allowlist is configured,
	// but AllowCredentials is only meaningful when origin is explicitly validated.
	config.AllowOriginFunc = func(origin string) bool { return true }
	config.AllowCredentials = false
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Authorization", "Content-Type", "Accept", "Origin", "X-Requested-With", "X-Api-Key", "X-Goog-Api-Key"}
	config.ExposeHeaders = []string{"Content-Length"}
	return cors.New(config)
}

func Version() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-New-Api-Version", common.Version)
		c.Next()
	}
}
