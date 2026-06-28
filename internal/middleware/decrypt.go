package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/Eressleep/metrics-tpl/internal/crypto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func DecryptMiddleware(privateKey *rsa.PrivateKey, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Encrypted") != "true" {
			c.Next()
			return
		}

		if privateKey == nil {
			logger.Warn("Received encrypted request but no private key configured")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "server not configured for encryption",
			})
			c.Abort()
			return
		}

		encryptedData, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logger.Error("Failed to read encrypted body", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "failed to read request body",
			})
			c.Abort()
			return
		}
		defer c.Request.Body.Close()

		decryptedData, err := crypto.DecryptRSA(encryptedData, privateKey)
		if err != nil {
			logger.Error("Failed to decrypt request body", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "failed to decrypt request body",
			})
			c.Abort()
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(decryptedData))
		c.Request.ContentLength = int64(len(decryptedData))

		logger.Debug("Request body decrypted successfully",
			zap.Int("size", len(decryptedData)))

		c.Next()
	}
}
