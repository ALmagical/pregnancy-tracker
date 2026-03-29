package api

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"pregnancy-tracker/server/pkg/resp"
)

func (s *Server) listArticles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	cat := c.Query("category")
	q := strings.TrimSpace(c.Query("q"))
	week := c.Query("week")
	if page < 1 {
		page = 1
	}
	if ps < 1 || ps > 50 {
		ps = 10
	}
	ctx := c.Request.Context()
	sql := `SELECT id, title, summary, cover, tags, read_minutes, published_at FROM articles WHERE 1=1`
	args := []interface{}{}
	n := 1
	if q != "" {
		sql += ` AND (title ILIKE $` + strconv.Itoa(n) + ` OR summary ILIKE $` + strconv.Itoa(n) + `)`
		args = append(args, "%"+q+"%")
		n++
	}
	if cat != "" {
		sql += ` AND $` + strconv.Itoa(n) + ` = ANY(tags)`
		args = append(args, cat)
		n++
	}
	if week != "" {
		_ = week
	}
	sql += ` ORDER BY published_at DESC`
	rows, err := s.Pool.Query(ctx, sql, args...)
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	defer rows.Close()
	var all []gin.H
	for rows.Next() {
		var id, title, summary, cover string
		var tags []string
		var rm int
		var pub interface{}
		if rows.Scan(&id, &title, &summary, &cover, &tags, &rm, &pub) != nil {
			continue
		}
		all = append(all, gin.H{
			"id": id, "title": title, "summary": summary, "cover": cover, "tags": tags, "readMinutes": rm,
			"publishedAt": pub,
		})
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
	slice := all[start:end]
	tp := (total + ps - 1) / ps
	if tp < 1 {
		tp = 1
	}
	resp.OK(c, gin.H{
		"list": slice,
		"pagination": gin.H{"page": page, "pageSize": ps, "total": total, "totalPages": tp},
	})
}

func (s *Server) getArticle(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	var title, summary, cover, content, source string
	var tags []string
	var rm int
	var pub interface{}
	err := s.Pool.QueryRow(ctx, `SELECT title, summary, cover, content, source, tags, read_minutes, published_at FROM articles WHERE id=$1`, id).Scan(
		&title, &summary, &cover, &content, &source, &tags, &rm, &pub)
	if err == pgx.ErrNoRows {
		resp.NotFound(c, "文章不存在")
		return
	}
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	related := []gin.H{}
	rows, _ := s.Pool.Query(ctx, `SELECT id, title FROM articles WHERE id <> $1 ORDER BY published_at DESC LIMIT 3`, id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var rid, rt string
			if rows.Scan(&rid, &rt) == nil {
				related = append(related, gin.H{"id": rid, "title": rt})
			}
		}
	}
	resp.OK(c, gin.H{
		"id": id, "title": title, "summary": summary, "content": content, "cover": cover,
		"source": source, "tags": tags, "publishedAt": pub, "readMinutes": rm,
		"related":     related,
		"disclaimer":  "以上内容仅供参考，如有不适请及时就医",
	})
}

func (s *Server) postFavorite(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	var body struct {
		TargetType string `json:"targetType"`
		TargetID   string `json:"targetId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.TargetType == "" || body.TargetID == "" {
		resp.BadRequest(c, "参数错误", "E_PARAM_INVALID", nil)
		return
	}
	ctx := c.Request.Context()
	_, err := s.Pool.Exec(ctx, `INSERT INTO favorites (user_id, target_type, target_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
		userID, body.TargetType, body.TargetID)
	if err != nil {
		resp.Internal(c, "收藏失败")
		return
	}
	resp.OK(c, gin.H{})
}

func (s *Server) deleteFavorite(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	tt := c.Query("targetType")
	tid := c.Query("targetId")
	if tt == "" || tid == "" {
		resp.BadRequest(c, "参数错误", "E_PARAM_INVALID", nil)
		return
	}
	ctx := c.Request.Context()
	_, _ = s.Pool.Exec(ctx, `DELETE FROM favorites WHERE user_id=$1 AND target_type=$2 AND target_id=$3`, userID, tt, tid)
	resp.OK(c, gin.H{})
}
