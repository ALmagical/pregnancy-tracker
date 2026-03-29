package api

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pregnancy-tracker/server/pkg/resp"
)

type checklistSeed struct {
	Cat, Title, Source string
}

var defaultChecklistRows = []checklistSeed{
	{"cat_docs", "身份证/医保卡", "template"},
	{"cat_docs", "产检资料册", "template"},
	{"cat_mom", "产妇卫生巾/产褥垫", "template"},
	{"cat_mom", "宽松睡衣与拖鞋", "template"},
	{"cat_baby", "新生儿衣物", "template"},
	{"cat_baby", "纸尿裤与湿巾", "template"},
	{"cat_other", "充电器与充电宝", "template"},
}

func (s *Server) insertDefaultChecklist(ctx context.Context, userID uuid.UUID) {
	for i, row := range defaultChecklistRows {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO checklist_items (user_id, category_id, title, checked, note, source, sort_order) VALUES ($1,$2,$3,false,'',$4,$5)`,
			userID, row.Cat, row.Title, row.Source, i)
	}
}

func (s *Server) getChecklist(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	ctx := c.Request.Context()
	var n int
	_ = s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_items WHERE user_id=$1`, userID).Scan(&n)
	if n == 0 {
		s.insertDefaultChecklist(ctx, userID)
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, category_id, title, checked, note, source, sort_order FROM checklist_items WHERE user_id=$1 ORDER BY sort_order, created_at`, userID)
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	defer rows.Close()

	catOrder := []struct{ id, title string }{
		{"cat_docs", "证件资料"},
		{"cat_mom", "妈妈用品"},
		{"cat_baby", "宝宝用品"},
		{"cat_other", "其他"},
		{"custom", "自定义"},
	}
	catMap := map[string][]gin.H{}
	order := []string{}
	for _, co := range catOrder {
		catMap[co.id] = []gin.H{}
		order = append(order, co.id)
	}

	done, total := 0, 0
	for rows.Next() {
		var id uuid.UUID
		var cat, title, note, source string
		var checked bool
		var sort int
		if rows.Scan(&id, &cat, &title, &checked, &note, &source, &sort) != nil {
			continue
		}
		total++
		if checked {
			done++
		}
		if _, ok := catMap[cat]; !ok {
			cat = "custom"
			if _, ok2 := catMap["custom"]; !ok2 {
				catMap["custom"] = []gin.H{}
				order = append(order, "custom")
			}
		}
		catMap[cat] = append(catMap[cat], gin.H{
			"id": id.String(), "title": title, "checked": checked, "note": note, "source": source,
		})
	}

	titleByID := map[string]string{"cat_docs": "证件资料", "cat_mom": "妈妈用品", "cat_baby": "宝宝用品", "cat_other": "其他", "custom": "自定义"}
	categories := []gin.H{}
	for _, cid := range order {
		items := catMap[cid]
		if len(items) == 0 {
			continue
		}
		categories = append(categories, gin.H{
			"id": cid, "title": titleByID[cid], "items": items,
		})
	}

	resp.OK(c, gin.H{
		"version":    1,
		"progress":   gin.H{"done": done, "total": total},
		"categories": categories,
	})
}

func (s *Server) putChecklistItem(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	id := c.Param("id")
	var body struct {
		Checked bool   `json:"checked"`
		Note    string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)
	ctx := c.Request.Context()
	res, err := s.Pool.Exec(ctx, `UPDATE checklist_items SET checked=$3, note=$4, updated_at=now() WHERE id=$1 AND user_id=$2`, id, userID, body.Checked, body.Note)
	if err != nil {
		resp.Internal(c, "更新失败")
		return
	}
	if res.RowsAffected() == 0 {
		resp.NotFound(c, "清单项不存在")
		return
	}
	resp.OK(c, gin.H{"id": id, "checked": body.Checked, "note": body.Note})
}

func (s *Server) postChecklistItem(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	var body struct {
		CategoryID string `json:"categoryId"`
		Title      string `json:"title"`
		Note       string `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.BadRequest(c, "参数错误", "E_PARAM_INVALID", nil)
		return
	}
	t := strings.TrimSpace(body.Title)
	n := strings.TrimSpace(body.Note)
	if t == "" {
		resp.BadRequest(c, "请填写清单项名称", "E_PARAM_INVALID", nil)
		return
	}
	if utf8.RuneCountInString(t) > 20 || utf8.RuneCountInString(n) > 50 {
		resp.BadRequest(c, "清单项长度超出限制", "E_CHECKLIST_ITEM_TOO_LONG", nil)
		return
	}
	cat := body.CategoryID
	if cat == "" {
		cat = "custom"
	}
	ctx := c.Request.Context()
	var maxSort int
	_ = s.Pool.QueryRow(ctx, `SELECT COALESCE(MAX(sort_order),0) FROM checklist_items WHERE user_id=$1`, userID).Scan(&maxSort)
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
INSERT INTO checklist_items (user_id, category_id, title, checked, note, source, sort_order)
VALUES ($1,$2,$3,false,$4,'custom',$5) RETURNING id`, userID, cat, t, n, maxSort+1).Scan(&id)
	if err != nil {
		resp.Internal(c, "添加失败")
		return
	}
	resp.OK(c, gin.H{"id": id.String(), "source": "custom"})
}

func (s *Server) resetChecklist(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	var body struct {
		KeepCustom bool `json:"keepCustomItems"`
	}
	_ = c.ShouldBindJSON(&body)
	ctx := c.Request.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		resp.Internal(c, "事务失败")
		return
	}
	defer tx.Rollback(ctx)
	if body.KeepCustom {
		_, err = tx.Exec(ctx, `DELETE FROM checklist_items WHERE user_id=$1 AND source='template'`, userID)
	} else {
		_, err = tx.Exec(ctx, `DELETE FROM checklist_items WHERE user_id=$1`, userID)
	}
	if err != nil {
		resp.Internal(c, "重置失败")
		return
	}
	for i, row := range defaultChecklistRows {
		_, _ = tx.Exec(ctx, `INSERT INTO checklist_items (user_id, category_id, title, checked, note, source, sort_order) VALUES ($1,$2,$3,false,'',$4,$5)`,
			userID, row.Cat, row.Title, row.Source, i)
	}
	if err := tx.Commit(ctx); err != nil {
		resp.Internal(c, "重置失败")
		return
	}
	resp.OK(c, gin.H{})
}
