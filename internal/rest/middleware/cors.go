package middleware

import "github.com/gin-gonic/gin"

// CORS will handle the CORS middleware
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

		if c.Request.Method == "OPTIONS" {
			c.Status(204)
			return
		}

		c.Next()
	}
}
