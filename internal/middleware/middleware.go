package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourorg/weather-api/internal/auth"
	"github.com/yourorg/weather-api/internal/logger"
	"go.uber.org/zap"
)

const headerRequestID = "X-Request-ID"
const headerAPIToken = "X-API-Token"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(headerRequestID)
		if id == "" {
			id = uuid.New().String()
		}
		c.Set("request_id", id)
		c.Header(headerRequestID, id)
		c.Next()
	}
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		log := logger.Get()
		requestID, _ := c.Get("request_id")

		log.Info("request",
			zap.String("request_id", requestID.(string)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log := logger.Get()
				requestID, _ := c.Get("request_id")
				log.Error("panic recovered",
					zap.Any("message", err),
					zap.String("request_id", requestID.(string)),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"message": "Ocorreu um erro interno no servidor!",
				})
			}
		}()
		c.Next()
	}
}

func StaticToken(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(headerAPIToken)
		if !auth.ValidateStaticToken(token, expectedToken) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "O Token informado é inválido ou expirou!",
			})
			return
		}
		c.Next()
	}
}
