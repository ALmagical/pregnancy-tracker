package api

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	"pregnancy-tracker/server/internal/timeutil"
	"pregnancy-tracker/server/pkg/resp"
)

func (s *Server) getUserInfo(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	ctx := c.Request.Context()
	var openid string
	var createdAt, updatedAt time.Time
	var status string
	var lastPeriod, dueDate sql.NullTime
	var preW, heightCm, curW sql.NullFloat64

	err := s.Pool.QueryRow(ctx, `
SELECT u.openid, u.created_at, u.updated_at,
       COALESCE(p.status,'pregnant'), p.last_period_date, p.due_date,
       p.pre_pregnancy_weight, p.height_cm, p.current_weight
FROM users u
LEFT JOIN user_profiles p ON p.user_id = u.id
WHERE u.id=$1`, userID).Scan(
		&openid, &createdAt, &updatedAt, &status, &lastPeriod, &dueDate, &preW, &heightCm, &curW)
	if err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}

	out := gin.H{
		"id":        userID.String(),
		"openid":    openid,
		"status":    status,
		"createdAt": createdAt.UTC().Format(time.RFC3339),
		"updatedAt": updatedAt.UTC().Format(time.RFC3339),
	}
	if lastPeriod.Valid {
		out["lastPeriodDate"] = lastPeriod.Time.Format("2006-01-02")
		lp := lastPeriod.Time
		due := timeutil.DueFromLMP(lp)
		days := timeutil.GestationalDays(lp)
		w, d := timeutil.WeekDayFromGestationalDays(days)
		out["dueDate"] = due.Format("2006-01-02")
		out["currentWeek"] = w
		out["currentDay"] = d
	}
	if dueDate.Valid && !lastPeriod.Valid {
		out["dueDate"] = dueDate.Time.Format("2006-01-02")
	}
	if preW.Valid {
		out["prePregnancyWeight"] = preW.Float64
	}
	if heightCm.Valid {
		out["heightCm"] = heightCm.Float64
	}
	if curW.Valid {
		out["currentWeight"] = curW.Float64
	}
	resp.OK(c, out)
}

func (s *Server) putUserInfo(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	var body struct {
		Status             *string  `json:"status"`
		LastPeriodDate     *string  `json:"lastPeriodDate"`
		PrePregnancyWeight *float64 `json:"prePregnancyWeight"`
		CurrentWeight      *float64 `json:"currentWeight"`
		HeightCm           *float64 `json:"heightCm"`
		Height             *float64 `json:"height"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.BadRequest(c, "参数错误", "E_PARAM_INVALID", nil)
		return
	}

	ctx := c.Request.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		resp.Internal(c, "事务失败")
		return
	}
	defer tx.Rollback(ctx)

	status := "pregnant"
	if body.Status != nil && *body.Status != "" {
		status = *body.Status
	}
	var lastPeriod sql.NullTime
	if body.LastPeriodDate != nil && *body.LastPeriodDate != "" {
		t, err := timeutil.ParseDate(*body.LastPeriodDate)
		if err != nil {
			resp.BadRequest(c, "末次月经日期无效", "E_PARAM_INVALID", nil)
			return
		}
		today := timeutil.TodayDate()
		if t.After(today) {
			resp.BadRequest(c, "末次月经日期不能晚于今天", "E_PARAM_INVALID", nil)
			return
		}
		if today.Sub(t).Hours()/24 > 365*2 {
			resp.BadRequest(c, "末次月经日期过早", "E_PARAM_INVALID", nil)
			return
		}
		lastPeriod = sql.NullTime{Time: t, Valid: true}
	}

	var preW, hcm, curW sql.NullFloat64
	if body.PrePregnancyWeight != nil {
		preW = sql.NullFloat64{Float64: *body.PrePregnancyWeight, Valid: true}
	}
	if body.CurrentWeight != nil {
		curW = sql.NullFloat64{Float64: *body.CurrentWeight, Valid: true}
	}
	if body.HeightCm != nil {
		hcm = sql.NullFloat64{Float64: *body.HeightCm, Valid: true}
	} else if body.Height != nil {
		hcm = sql.NullFloat64{Float64: *body.Height, Valid: true}
	}

	var dueDate sql.NullTime
	if lastPeriod.Valid {
		d := timeutil.DueFromLMP(lastPeriod.Time)
		dueDate = sql.NullTime{Time: d, Valid: true}
	}

	_, err = tx.Exec(ctx, `INSERT INTO user_profiles (user_id, status, last_period_date, due_date, pre_pregnancy_weight, height_cm, current_weight, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,now())
ON CONFLICT (user_id) DO UPDATE SET
  status=EXCLUDED.status,
  last_period_date=COALESCE(EXCLUDED.last_period_date, user_profiles.last_period_date),
  due_date=COALESCE(EXCLUDED.due_date, user_profiles.due_date),
  pre_pregnancy_weight=COALESCE(EXCLUDED.pre_pregnancy_weight, user_profiles.pre_pregnancy_weight),
  height_cm=COALESCE(EXCLUDED.height_cm, user_profiles.height_cm),
  current_weight=COALESCE(EXCLUDED.current_weight, user_profiles.current_weight),
  updated_at=now()`,
		userID, status, nullTime(lastPeriod), nullTime(dueDate), nullFloat(preW), nullFloat(hcm), nullFloat(curW))
	if err != nil {
		resp.Internal(c, "保存失败")
		return
	}
	_, _ = tx.Exec(ctx, `UPDATE users SET updated_at=now() WHERE id=$1`, userID)
	if err := tx.Commit(ctx); err != nil {
		resp.Internal(c, "保存失败")
		return
	}

	// 读取计算孕周
	var lpOut sql.NullTime
	_ = s.Pool.QueryRow(ctx, `SELECT last_period_date FROM user_profiles WHERE user_id=$1`, userID).Scan(&lpOut)
	data := gin.H{"message": "更新成功"}
	if lpOut.Valid {
		days := timeutil.GestationalDays(lpOut.Time)
		w, d := timeutil.WeekDayFromGestationalDays(days)
		data["dueDate"] = timeutil.DueFromLMP(lpOut.Time).Format("2006-01-02")
		data["currentWeek"] = w
		data["currentDay"] = d
	}
	resp.OK(c, data)
}

func nullTime(nt sql.NullTime) interface{} {
	if !nt.Valid {
		return nil
	}
	return nt.Time
}

func nullFloat(nf sql.NullFloat64) interface{} {
	if !nf.Valid {
		return nil
	}
	return nf.Float64
}
