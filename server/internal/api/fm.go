package api

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"pregnancy-tracker/server/pkg/resp"
)

func (s *Server) fmCreateSession(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	var body struct {
		StartedAt string `json:"startedAt"`
		Note      string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)
	started := body.StartedAt
	if started == "" {
		started = time.Now().UTC().Format(time.RFC3339)
	}
	st, err := time.Parse(time.RFC3339, started)
	if err != nil {
		resp.BadRequest(c, "时间格式无效", "E_PARAM_INVALID", nil)
		return
	}
	ctx := c.Request.Context()
	var id uuid.UUID
	err = s.Pool.QueryRow(ctx, `
INSERT INTO fm_sessions (user_id, started_at, status, count, note) VALUES ($1,$2,'running',0,$3) RETURNING id`,
		userID, st, body.Note).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			resp.Conflict(c, "已有未结束的胎动计数", "E_FM_SESSION_RUNNING")
			return
		}
		resp.Internal(c, "创建失败")
		return
	}
	resp.OK(c, gin.H{"id": id.String(), "startedAt": st.UTC().Format(time.RFC3339), "status": "running"})
}

func (s *Server) fmEvent(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	sid := c.Param("id")
	var body struct {
		Type string `json:"type"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || (body.Type != "add" && body.Type != "undo") {
		resp.BadRequest(c, "type 须为 add 或 undo", "E_PARAM_INVALID", nil)
		return
	}
	ctx := c.Request.Context()
	var cnt int
	var startedAt time.Time
	err := s.Pool.QueryRow(ctx, `SELECT count, started_at FROM fm_sessions WHERE id=$1 AND user_id=$2 AND status='running'`, sid, userID).Scan(&cnt, &startedAt)
	if err == pgx.ErrNoRows {
		resp.NotFound(c, "会话不存在或已结束")
		return
	}
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	if body.Type == "add" {
		cnt++
		if cnt > 999 {
			cnt = 999
		}
	} else {
		cnt--
		if cnt < 0 {
			cnt = 0
		}
	}
	_, err = s.Pool.Exec(ctx, `UPDATE fm_sessions SET count=$3, updated_at=now() WHERE id=$1 AND user_id=$2`, sid, userID, cnt)
	if err != nil {
		resp.Internal(c, "更新失败")
		return
	}
	resp.OK(c, gin.H{
		"sessionId":   sid,
		"count":       cnt,
		"lastEventAt": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) fmFinish(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	sid := c.Param("id")
	var body struct {
		EndedAt   string `json:"endedAt"`
		ResultTag string `json:"resultTag"`
	}
	_ = c.ShouldBindJSON(&body)
	ended := body.EndedAt
	if ended == "" {
		ended = time.Now().UTC().Format(time.RFC3339)
	}
	et, err := time.Parse(time.RFC3339, ended)
	if err != nil {
		resp.BadRequest(c, "时间无效", "E_PARAM_INVALID", nil)
		return
	}
	ctx := c.Request.Context()
	var startedAt time.Time
	var cnt int
	err = s.Pool.QueryRow(ctx, `SELECT started_at, count FROM fm_sessions WHERE id=$1 AND user_id=$2 AND status='running'`, sid, userID).Scan(&startedAt, &cnt)
	if err == pgx.ErrNoRows {
		resp.NotFound(c, "没有进行中的会话")
		return
	}
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	dur := int(et.Sub(startedAt).Seconds())
	if dur < 0 {
		dur = 0
	}
	_, err = s.Pool.Exec(ctx, `UPDATE fm_sessions SET status='finished', ended_at=$3, count=$4, result_tag=$5, updated_at=now() WHERE id=$1 AND user_id=$2`,
		sid, userID, et, cnt, body.ResultTag)
	if err != nil {
		resp.Internal(c, "保存失败")
		return
	}
	resp.OK(c, gin.H{"id": sid, "status": "finished", "count": cnt, "durationSec": dur})
}

func (s *Server) fmListSessions(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	ctx := c.Request.Context()
	rows, err := s.Pool.Query(ctx, `
SELECT id, started_at, ended_at, count,
  CASE WHEN ended_at IS NOT NULL AND started_at IS NOT NULL
    THEN EXTRACT(EPOCH FROM (ended_at - started_at))::int ELSE 0 END AS dur
FROM fm_sessions
WHERE user_id=$1 AND status='finished'
ORDER BY started_at DESC
LIMIT $2`, userID, limit)
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var st, en sql.NullTime
		var cnt, dur int
		if err := rows.Scan(&id, &st, &en, &cnt, &dur); err != nil {
			continue
		}
		item := gin.H{"id": id.String(), "count": cnt, "durationSec": dur, "syncStatus": "synced"}
		if st.Valid {
			item["startedAt"] = st.Time.UTC().Format(time.RFC3339)
		}
		if en.Valid {
			item["endedAt"] = en.Time.UTC().Format(time.RFC3339)
		}
		list = append(list, item)
	}
	resp.OK(c, gin.H{"list": list})
}

func (s *Server) fmSummary(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	start := c.Query("startDate")
	end := c.Query("endDate")
	ctx := c.Request.Context()
	q := `
SELECT (started_at AT TIME ZONE 'Asia/Shanghai')::date AS d,
       SUM(count)::int AS total,
       COUNT(*)::int AS sessions
FROM fm_sessions
WHERE user_id=$1 AND status='finished'`
	args := []interface{}{userID}
	if start != "" {
		q += ` AND (started_at AT TIME ZONE 'Asia/Shanghai')::date >= $` + strconv.Itoa(len(args)+1)
		args = append(args, start)
	}
	if end != "" {
		q += ` AND (started_at AT TIME ZONE 'Asia/Shanghai')::date <= $` + strconv.Itoa(len(args)+1)
		args = append(args, end)
	}
	q += ` GROUP BY 1 ORDER BY 1 DESC`
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var d time.Time
		var total, sc int
		if rows.Scan(&d, &total, &sc) != nil {
			continue
		}
		list = append(list, gin.H{
			"date":         d.Format("2006-01-02"),
			"totalCount":   total,
			"sessionCount": sc,
		})
	}
	resp.OK(c, gin.H{"list": list})
}
