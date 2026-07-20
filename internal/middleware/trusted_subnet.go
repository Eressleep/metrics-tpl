package middleware

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TrustedSubnetMiddleware проверяет, что IP-адрес клиента входит в доверенную подсеть
func TrustedSubnetMiddleware(trustedSubnet string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Если доверенная подсеть не указана, пропускаем все запросы
		if trustedSubnet == "" {
			c.Next()
			return
		}

		realIP := c.GetHeader("X-Real-IP")
		if realIP == "" {
			logger.Warn("X-Real-IP header is missing",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method))
			c.JSON(http.StatusForbidden, gin.H{
				"error": "X-Real-IP header is required",
			})
			c.Abort()
			return
		}

		// Парсим доверенную подсеть
		_, ipNet, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			logger.Error("Invalid trusted subnet CIDR",
				zap.String("trusted_subnet", trustedSubnet),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "invalid trusted subnet configuration",
			})
			c.Abort()
			return
		}

		// Парсим IP-адрес из заголовка
		ip := net.ParseIP(realIP)
		if ip == nil {
			logger.Warn("Invalid IP address in X-Real-IP header",
				zap.String("x-real-ip", realIP),
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusForbidden, gin.H{
				"error": "invalid IP address",
			})
			c.Abort()
			return
		}

		if !ipNet.Contains(ip) {
			logger.Warn("IP address not in trusted subnet",
				zap.String("ip", realIP),
				zap.String("trusted_subnet", trustedSubnet),
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusForbidden, gin.H{
				"error": "IP address not in trusted subnet",
			})
			c.Abort()
			return
		}

		logger.Debug("IP address verified",
			zap.String("ip", realIP),
			zap.String("trusted_subnet", trustedSubnet))

		c.Next()
	}
}
