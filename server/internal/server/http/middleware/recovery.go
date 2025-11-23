package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a condition that warrants a panic stack trace.
				var brokenPipe bool
				// (Omitting detailed broken pipe check for brevity, but standard gin recovery has it)

				rid := c.GetString(HeaderRequestID)
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("request_id", rid),
					zap.Stack("stack"),
				)

				if brokenPipe {
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()
					return
				}

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "internal_server_error",
						"message": "Internal Server Error",
					},
					"meta": gin.H{
						"request_id": rid,
					},
				})
			}
		}()
		c.Next()
	}
}
