package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// SetRequestContextWithTimeout will set the request context with timeout for every incoming HTTP Request
func SetRequestContextWithTimeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
