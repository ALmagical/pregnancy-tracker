package api

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pregnancy-tracker/server/internal/timeutil"
	"pregnancy-tracker/server/pkg/resp"
)

func (s *Server) listContractions(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	dateFilter := c.Query("date")
	ctx := c.Request.Context()
	q := `
SELECT id, started_at, ended_at, duration_sec, COALESCE(interval_sec,0)
FROM contractions WHERE user_id=$1`
	args := []interface{}{userID}
	if dateFilter != "" {
		q += ` AND (started_at AT TIME ZONE 'Asia/Shanghai')::date = $2::date`
		args = append(args, dateFilter)
	}
	q += ` ORDER BY started_at DESC`
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var st, en time.Time
		var dur, iv int
		if rows.Scan(&id, &st, &en, &dur, &iv) != nil {
			continue
		}
		list = append(list, gin.H{
			"id":          id.String(),
			"startedAt":   st.UTC().Format(time.RFC3339),
			"endedAt":     en.UTC().Format(time.RFC3339),
			"durationSec": dur,
			"intervalSec": iv,
		})
	}
	resp.OK(c, gin.H{"list": list})
}

func (s *Server) createContraction(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	var body struct {
		StartedAt string `json:"startedAt"`
		EndedAt   string `json:"endedAt"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.StartedAt == "" || body.EndedAt == "" {
		resp.BadRequest(c, "请填写开始与结束时间", "E_PARAM_INVALID", nil)
		return
	}
	st, err1 := time.Parse(time.RFC3339, body.StartedAt)
	en, err2 := time.Parse(time.RFC3339, body.EndedAt)
	if err1 != nil || err2 != nil || en.Before(st) {
		resp.BadRequest(c, "结束时间不能早于开始时间", "E_CONTRACTION_TIME_INVALID", nil)
		return
	}
	dur := int(en.Sub(st).Seconds())
	if dur < 5 {
		// 仍保存，与设计「提示」一致
	}
	if dur > 600 {
		// 10 分钟以上仍保存
	}

	ctx := c.Request.Context()
	day := st.In(timeutil.Shanghai())
	dayStr := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, timeutil.Shanghai()).Format("2006-01-02")

	var nt sql.NullTime
	_ = s.Pool.QueryRow(ctx, `
SELECT started_at FROM contractions
WHERE user_id=$1 AND (started_at AT TIME ZONE 'Asia/Shanghai')::date = $2::date
  AND started_at < $3
ORDER BY started_at DESC LIMIT 1`, userID, dayStr, st).Scan(&nt)

	var intervalSec interface{}
	if nt.Valid {
		sec := int(st.Sub(nt.Time).Seconds())
		if sec < 0 {
			sec = 0
		}
		intervalSec = sec
	}

	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
INSERT INTO contractions (user_id, started_at, ended_at, duration_sec, interval_sec)
VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		userID, st, en, dur, intervalSec).Scan(&id)
	if err != nil {
		resp.Internal(c, "保存失败")
		return
	}
	out := gin.H{"id": id.String(), "durationSec": dur, "intervalSec": 0}
	if v, ok := intervalSec.(int); ok {
		out["intervalSec"] = v
	}
	resp.OK(c, out)
}
