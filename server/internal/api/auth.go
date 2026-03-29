package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"pregnancy-tracker/server/internal/auth"
	"pregnancy-tracker/server/internal/wechat"
	"pregnancy-tracker/server/pkg/resp"
)

type wechatLoginBody struct {
	Code string `json:"code"`
}

func (s *Server) postAuthWechat(c *gin.Context) {
	var body wechatLoginBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Code == "" {
		resp.BadRequest(c, "缺少 code", "E_PARAM_INVALID", nil)
		return
	}

	var openID string
	if s.Cfg.WeChatMock {
		openID = "mock_" + body.Code
		if len(openID) > 64 {
			openID = openID[:64]
		}
	} else {
		if s.Cfg.WeChatAppID == "" || s.Cfg.WeChatAppSecret == "" {
			resp.Internal(c, "服务端未配置微信 AppId/Secret")
			return
		}
		sess, err := wechat.Code2Session(s.Cfg.WeChatAppID, s.Cfg.WeChatAppSecret, body.Code)
		if err != nil {
			resp.BadRequest(c, "微信登录失败", "E_PARAM_INVALID", map[string]interface{}{"detail": err.Error()})
			return
		}
		openID = sess.OpenID
	}

	ctx := c.Request.Context()
	var userID uuid.UUID
	err := s.Pool.QueryRow(ctx, `SELECT id FROM users WHERE openid=$1`, openID).Scan(&userID)
	if err == pgx.ErrNoRows {
		err = s.Pool.QueryRow(ctx,
			`INSERT INTO users (openid) VALUES ($1) RETURNING id`, openID).Scan(&userID)
	}
	if err != nil {
		resp.Internal(c, "创建用户失败")
		return
	}

	exp := time.Duration(s.Cfg.JWTExpireHours) * time.Hour
	if exp <= 0 {
		exp = 30 * 24 * time.Hour
	}
	token, err := auth.Sign(userID, s.Cfg.JWTSecret, exp)
	if err != nil {
		resp.Internal(c, "签发令牌失败")
		return
	}

	resp.OK(c, gin.H{
		"accessToken": token,
		"token":       token,
		"expiresIn":   int(exp.Seconds()),
	})
}
