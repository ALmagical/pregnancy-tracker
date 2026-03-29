package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"pregnancy-tracker/server/internal/auth"
	"pregnancy-tracker/server/internal/config"
	"pregnancy-tracker/server/pkg/resp"
)

const CtxUserID = "userID"

func JWT(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			resp.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}
		raw := strings.TrimSpace(h[7:])
		cl, err := auth.Parse(raw, cfg.JWTSecret)
		if err != nil {
			resp.Unauthorized(c, "登录已失效")
			c.Abort()
			return
		}
		c.Set(CtxUserID, cl.UserID)
		c.Next()
	}
}
