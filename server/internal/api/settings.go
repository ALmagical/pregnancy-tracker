package api

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"pregnancy-tracker/server/pkg/resp"
)

var defaultPush = map[string]interface{}{
	"dailyKnowledge":        true,
	"checkupReminder":       true,
	"weightReminder":        false,
	"fetalMovementReminder": false,
	"latePregnancyReminder": true,
	"quietHours":            map[string]interface{}{"enabled": true, "start": "22:00", "end": "07:00"},
}
var defaultAI = map[string]interface{}{"contextEnabled": true}

func (s *Server) getSettings(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	ctx := c.Request.Context()
	var push, ai []byte
	err := s.Pool.QueryRow(ctx, `SELECT push, ai FROM user_settings WHERE user_id=$1`, userID).Scan(&push, &ai)
	if err == pgx.ErrNoRows {
		resp.OK(c, gin.H{"push": defaultPush, "ai": defaultAI})
		return
	}
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	var pushObj, aiObj map[string]interface{}
	_ = json.Unmarshal(push, &pushObj)
	_ = json.Unmarshal(ai, &aiObj)
	resp.OK(c, gin.H{
		"push": mergeMap(defaultPush, pushObj),
		"ai":   mergeMap(defaultAI, aiObj),
	})
}

func mergeMap(base, over map[string]interface{}) map[string]interface{} {
	m := map[string]interface{}{}
	for k, v := range base {
		m[k] = v
	}
	if over != nil {
		for k, v := range over {
			m[k] = v
		}
	}
	return m
}

func (s *Server) putSettings(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.BadRequest(c, "参数错误", "E_PARAM_INVALID", nil)
		return
	}
	pushB, _ := json.Marshal(body["push"])
	aiB, _ := json.Marshal(body["ai"])
	if len(pushB) < 3 {
		pushB = []byte("{}")
	}
	if len(aiB) < 3 {
		aiB = []byte("{}")
	}
	ctx := c.Request.Context()
	_, err := s.Pool.Exec(ctx, `
INSERT INTO user_settings (user_id, push, ai, updated_at) VALUES ($1,$2::jsonb,$3::jsonb,now())
ON CONFLICT (user_id) DO UPDATE SET push=user_settings.push || EXCLUDED.push, ai=user_settings.ai || EXCLUDED.ai, updated_at=now()`,
		userID, pushB, aiB)
	if err != nil {
		resp.Internal(c, "保存失败")
		return
	}
	resp.OK(c, gin.H{})
}
