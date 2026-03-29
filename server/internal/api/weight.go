package api

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pregnancy-tracker/server/internal/timeutil"
	"pregnancy-tracker/server/pkg/resp"
)

func (s *Server) listWeights(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("pageSize", "30"))
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	if page < 1 {
		page = 1
	}
	if ps < 1 || ps > 200 {
		ps = 30
	}
	ctx := c.Request.Context()

	q := `SELECT id, weight, recorded_at, week, day FROM weights WHERE user_id=$1`
	args := []interface{}{userID}
	n := 2
	if startDate != "" {
		q += ` AND recorded_at >= $` + strconv.Itoa(n)
		args = append(args, startDate)
		n++
	}
	if endDate != "" {
		q += ` AND recorded_at <= $` + strconv.Itoa(n)
		args = append(args, endDate)
		n++
	}
	q += ` ORDER BY recorded_at DESC, created_at DESC`

	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	defer rows.Close()
	type wrow struct {
		id   string
		w    float64
		rd   time.Time
		week sql.NullInt32
		day  sql.NullInt32
	}
	var all []wrow
	for rows.Next() {
		var r wrow
		var wid uuid.UUID
		if err := rows.Scan(&wid, &r.w, &r.rd, &r.week, &r.day); err != nil {
			continue
		}
		r.id = wid.String()
		all = append(all, r)
	}
	total := len(all)
	start := (page - 1) * ps
	end := start + ps
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	list := make([]gin.H, 0, end-start)
	for _, r := range all[start:end] {
		item := gin.H{
			"id":         r.id,
			"weight":     r.w,
			"recordedAt": r.rd.Format("2006-01-02") + "T00:00:00Z",
		}
		if r.week.Valid {
			item["week"] = int(r.week.Int32)
		}
		if r.day.Valid {
			item["day"] = int(r.day.Int32)
		}
		list = append(list, item)
	}

	var preW, curW sql.NullFloat64
	_ = s.Pool.QueryRow(ctx, `SELECT pre_pregnancy_weight, current_weight FROM user_profiles WHERE user_id=$1`, userID).Scan(&preW, &curW)
	if !curW.Valid && len(all) > 0 {
		curW = sql.NullFloat64{Float64: all[0].w, Valid: true}
	}
	stats := gin.H{
		"prePregnancyWeight": nil,
		"currentWeight":      nil,
		"totalGain":          nil,
		"averageWeeklyGain":  nil,
		"recommendedRange":   gin.H{"min": 5.5, "max": 9.0},
	}
	if preW.Valid {
		stats["prePregnancyWeight"] = preW.Float64
	}
	if curW.Valid {
		stats["currentWeight"] = curW.Float64
	}
	if preW.Valid && curW.Valid {
		stats["totalGain"] = curW.Float64 - preW.Float64
	}

	resp.OK(c, gin.H{"list": list, "statistics": stats})
}

func (s *Server) createWeight(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	var body struct {
		Weight     float64 `json:"weight"`
		RecordedAt string  `json:"recordedAt"`
		Note       string  `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.BadRequest(c, "参数错误", "E_PARAM_INVALID", nil)
		return
	}
	if body.Weight < 1 || body.Weight > 500 {
		resp.BadRequest(c, "体重数值异常", "E_PARAM_INVALID", nil)
		return
	}
	rd := body.RecordedAt
	if len(rd) >= 10 {
		rd = rd[:10]
	} else {
		rd = timeutil.TodayDate().Format("2006-01-02")
	}
	if _, err := timeutil.ParseDate(rd); err != nil {
		resp.BadRequest(c, "日期无效", "E_PARAM_INVALID", nil)
		return
	}

	ctx := c.Request.Context()
	var week, day sql.NullInt32
	if lp := s.getLastPeriod(ctx, userID); lp != nil {
		days := timeutil.GestationalDays(*lp)
		w, d := timeutil.WeekDayFromGestationalDays(days)
		week = sql.NullInt32{Int32: int32(w), Valid: true}
		day = sql.NullInt32{Int32: int32(d), Valid: true}
	}

	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
INSERT INTO weights (user_id, weight, recorded_at, note, week, day) VALUES ($1,$2,$3::date,$4,$5,$6) RETURNING id`,
		userID, body.Weight, rd, body.Note, nullInt32(week), nullInt32(day)).Scan(&id)
	if err != nil {
		resp.Internal(c, "保存失败")
		return
	}
	_, _ = s.Pool.Exec(ctx, `UPDATE user_profiles SET current_weight=$2, updated_at=now() WHERE user_id=$1`, userID, body.Weight)
	resp.OK(c, gin.H{"id": id.String()})
}

func nullInt32(n sql.NullInt32) interface{} {
	if !n.Valid {
		return nil
	}
	return n.Int32
}

func (s *Server) getLastPeriod(ctx context.Context, userID uuid.UUID) *time.Time {
	var t sql.NullTime
	if err := s.Pool.QueryRow(ctx, `SELECT last_period_date FROM user_profiles WHERE user_id=$1`, userID).Scan(&t); err != nil || !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}

func (s *Server) deleteWeight(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	id := c.Param("id")
	ctx := c.Request.Context()
	res, err := s.Pool.Exec(ctx, `DELETE FROM weights WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		resp.Internal(c, "删除失败")
		return
	}
	if res.RowsAffected() == 0 {
		resp.NotFound(c, "记录不存在")
		return
	}
	resp.OK(c, gin.H{})
}
