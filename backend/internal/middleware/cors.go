package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS libera o frontend (servido separadamente, sem build step) para
// consumir a API. A origem permitida é configurável via ALLOWED_ORIGIN
// no .env; em produção deve ser o domínio real do frontend, nunca "*"
// quando cookies/credenciais estiverem envolvidos.
func CORS(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
