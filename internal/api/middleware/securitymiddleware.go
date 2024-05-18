package middleware

import (
	"github.com/gin-gonic/gin"
)

type SecurityMiddleware struct{}

func (s *SecurityMiddleware) RequestHeaders() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Header("X-Content-Type-Options", "nosniff")
		context.Next()
	}
}
