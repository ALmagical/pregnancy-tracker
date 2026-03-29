package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"pregnancy-tracker/server/internal/content"
	"pregnancy-tracker/server/internal/timeutil"
	"pregnancy-tracker/server/pkg/resp"
)

func (s *Server) getPregnancyWeek(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	wk, _ := strconv.Atoi(c.Param("week"))
	if wk < 1 {
		wk = 1
	}
	if wk > 42 {
		wk = 42
	}
	ctx := c.Request.Context()
	rows, err := s.Pool.Query(ctx, `SELECT task_id, done FROM pregnancy_week_tasks WHERE user_id=$1 AND week=$2`, userID, wk)
	tasksDone := map[string]bool{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tid string
			var done bool
			if rows.Scan(&tid, &done) == nil {
				tasksDone[tid] = done
			}
		}
	}

	knowledge := []map[string]interface{}{}
	arows, err := s.Pool.Query(ctx, `SELECT id, title, cover, tags, read_minutes FROM articles ORDER BY published_at DESC LIMIT 3`)
	if err == nil {
		defer arows.Close()
		for arows.Next() {
			var id, title, cover string
			var tags []string
			var rm int
			if arows.Scan(&id, &title, &cover, &tags, &rm) != nil {
				continue
			}
			knowledge = append(knowledge, map[string]interface{}{
				"id": id, "title": title, "cover": cover, "tags": tags, "readMinutes": rm,
			})
		}
	}

	day := 0
	if lp := s.getLastPeriod(ctx, userID); lp != nil {
		days := timeutil.GestationalDays(*lp)
		cw, cd := timeutil.WeekDayFromGestationalDays(days)
		if cw == wk {
			day = cd
		}
	}

	payload := content.WeekPayload(wk, day, tasksDone, knowledge)
	resp.OK(c, payload)
}

func (s *Server) putPregnancyTask(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	taskID := c.Param("taskId")
	var body struct {
		Done bool `json:"done"`
	}
	_ = c.ShouldBindJSON(&body)
	ctx := c.Request.Context()
	wk := 24
	if lp := s.getLastPeriod(ctx, userID); lp != nil {
		days := timeutil.GestationalDays(*lp)
		wk, _ = timeutil.WeekDayFromGestationalDays(days)
	}
	_, err := s.Pool.Exec(ctx, `
INSERT INTO pregnancy_week_tasks (user_id, week, task_id, done) VALUES ($1,$2,$3,$4)
ON CONFLICT (user_id, week, task_id) DO UPDATE SET done=EXCLUDED.done`,
		userID, wk, taskID, body.Done)
	if err != nil {
		resp.Internal(c, "保存失败")
		return
	}
	resp.OK(c, gin.H{"taskId": taskID, "done": body.Done})
}
