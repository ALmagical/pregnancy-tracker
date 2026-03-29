package api

import (
	"database/sql"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"pregnancy-tracker/server/internal/timeutil"
	"pregnancy-tracker/server/pkg/resp"
)

func (s *Server) listCheckups(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if ps < 1 || ps > 100 {
		ps = 10
	}
	today := timeutil.TodayDate()
	ctx := c.Request.Context()

	rows, err := s.Pool.Query(ctx, `
SELECT c.id, c.checkup_date, c.checkup_type, c.hospital, c.status,
       (SELECT COUNT(*)::int FROM checkup_reports r WHERE r.checkup_id=c.id) AS rc
FROM checkups c
WHERE c.user_id=$1
ORDER BY c.checkup_date DESC
`, userID)
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	defer rows.Close()

	type row struct {
		id   uuid.UUID
		typ, hosp, st string
		date              time.Time
		rc                int
	}
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.date, &r.typ, &r.hosp, &r.st, &r.rc); err != nil {
			continue
		}
		all = append(all, r)
	}

	statusFilter := c.Query("status")
	filtered := all
	if statusFilter != "" && statusFilter != "all" {
		var nf []row
		for _, r := range all {
			st := deriveCheckupStatus(r.date, today, r.st)
			if st == statusFilter {
				nf = append(nf, r)
			}
		}
		filtered = nf
	}

	total := len(filtered)
	start := (page - 1) * ps
	end := start + ps
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	slice := filtered[start:end]

	list := make([]gin.H, 0, len(slice))
	for _, r := range slice {
		st := deriveCheckupStatus(r.date, today, r.st)
		var sum sql.NullString
		_ = s.Pool.QueryRow(ctx, `SELECT summary FROM checkups WHERE id=$1`, r.id).Scan(&sum)
		item := gin.H{
			"id":          r.id.String(),
			"checkupDate": r.date.Format("2006-01-02"),
			"checkupType": r.typ,
			"hospital":    r.hosp,
			"status":      st,
			"hasReport":   r.rc > 0,
			"reportCount": r.rc,
		}
		if sum.Valid && sum.String != "" {
			item["summary"] = sum.String
		}
		list = append(list, item)
	}

	totalPages := (total + ps - 1) / ps
	if totalPages < 1 {
		totalPages = 1
	}
	resp.OK(c, gin.H{
		"list": list,
		"pagination": gin.H{
			"page": page, "pageSize": ps, "total": total, "totalPages": totalPages,
		},
	})
}

func deriveCheckupStatus(checkupDate, today time.Time, stored string) string {
	d0 := time.Date(checkupDate.Year(), checkupDate.Month(), checkupDate.Day(), 0, 0, 0, 0, checkupDate.Location())
	t0 := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	if stored == "completed" {
		return "completed"
	}
	if d0.Before(t0) {
		return "completed"
	}
	if d0.Equal(t0) {
		return "pending"
	}
	return "upcoming"
}

func (s *Server) getCheckup(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	id := c.Param("id")
	ctx := c.Request.Context()
	var cd time.Time
	var typ, tid, hosp, note, summary string
	var createdAt, updatedAt time.Time
	err := s.Pool.QueryRow(ctx, `
SELECT checkup_date, checkup_type, checkup_type_id, hospital, note, summary, created_at, updated_at
FROM checkups WHERE id=$1 AND user_id=$2`, id, userID).Scan(
		&cd, &typ, &tid, &hosp, &note, &summary, &createdAt, &updatedAt)
	if err == pgx.ErrNoRows {
		resp.NotFound(c, "记录不存在")
		return
	}
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}

	rows, err := s.Pool.Query(ctx, `SELECT id, public_url, COALESCE(thumb_url, public_url) FROM checkup_reports WHERE checkup_id=$1 ORDER BY sort_order, created_at`, id)
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	defer rows.Close()
	imgs := []gin.H{}
	for rows.Next() {
		var rid, url, thumb string
		if rows.Scan(&rid, &url, &thumb) == nil {
			imgs = append(imgs, gin.H{"id": rid, "url": url, "thumbnail": thumb})
		}
	}

	resp.OK(c, gin.H{
		"id":            id,
		"checkupDate":   cd.Format("2006-01-02"),
		"checkupType":   typ,
		"hospital":      hosp,
		"checkupTypeId": tid,
		"summary":       summary,
		"note":          note,
		"images":        imgs,
		"createdAt":     createdAt.UTC().Format(time.RFC3339),
		"updatedAt":     updatedAt.UTC().Format(time.RFC3339),
	})
}

func (s *Server) createCheckup(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	var body struct {
		CheckupDate     string `json:"checkupDate"`
		CheckupType     string `json:"checkupType"`
		CheckupTypeID   string `json:"checkupTypeId"`
		Hospital        string `json:"hospital"`
		Note            string `json:"note"`
		ClientRequestID string `json:"clientRequestId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.CheckupDate == "" || strings.TrimSpace(body.CheckupType) == "" {
		resp.BadRequest(c, "请填写产检日期与类型", "E_PARAM_INVALID", nil)
		return
	}
	cd, err := timeutil.ParseDate(body.CheckupDate)
	if err != nil {
		resp.BadRequest(c, "日期无效", "E_PARAM_INVALID", nil)
		return
	}
	today := timeutil.TodayDate()
	st := "pending"
	if cd.Before(today) {
		st = "completed"
	} else if cd.After(today) {
		st = "upcoming"
	}

	ctx := c.Request.Context()
	var id uuid.UUID
	err = s.Pool.QueryRow(ctx, `
INSERT INTO checkups (user_id, checkup_date, checkup_type, checkup_type_id, hospital, note, status)
VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		userID, cd, strings.TrimSpace(body.CheckupType), body.CheckupTypeID, body.Hospital, body.Note, st).Scan(&id)
	if err != nil {
		resp.Internal(c, "创建失败")
		return
	}
	resp.OK(c, gin.H{"id": id.String(), "reminderId": fmt.Sprintf("r_%s", id.String()[:8])})
}

func (s *Server) updateCheckup(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	id := c.Param("id")
	var body struct {
		CheckupDate   string `json:"checkupDate"`
		CheckupType   string `json:"checkupType"`
		CheckupTypeID string `json:"checkupTypeId"`
		Hospital      string `json:"hospital"`
		Note          string `json:"note"`
		Summary       string `json:"summary"`
		Status        string `json:"status"`
	}
	_ = c.ShouldBindJSON(&body)
	ctx := c.Request.Context()
	var cd time.Time
	var typ, tid, hosp, note, summary, st string
	err := s.Pool.QueryRow(ctx, `SELECT checkup_date, checkup_type, checkup_type_id, hospital, note, summary, status FROM checkups WHERE id=$1 AND user_id=$2`, id, userID).Scan(
		&cd, &typ, &tid, &hosp, &note, &summary, &st)
	if err == pgx.ErrNoRows {
		resp.NotFound(c, "记录不存在")
		return
	}
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	if body.CheckupDate != "" {
		if t, err := timeutil.ParseDate(body.CheckupDate); err == nil {
			cd = t
		}
	}
	if body.CheckupType != "" {
		typ = strings.TrimSpace(body.CheckupType)
	}
	if body.CheckupTypeID != "" {
		tid = body.CheckupTypeID
	}
	if body.Hospital != "" {
		hosp = body.Hospital
	}
	if body.Note != "" {
		note = body.Note
	}
	if body.Summary != "" {
		summary = body.Summary
	}
	if body.Status != "" {
		st = body.Status
	}
	today := timeutil.TodayDate()
	if st != "completed" && st != "pending" && st != "upcoming" {
		st = deriveCheckupStatus(cd, today, st)
	}
	res, err := s.Pool.Exec(ctx, `UPDATE checkups SET checkup_date=$3, checkup_type=$4, checkup_type_id=$5, hospital=$6, note=$7, summary=$8, status=$9, updated_at=now() WHERE id=$1 AND user_id=$2`,
		id, userID, cd, typ, tid, hosp, note, summary, st)
	if err != nil {
		resp.Internal(c, "更新失败")
		return
	}
	if res.RowsAffected() == 0 {
		resp.NotFound(c, "记录不存在")
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (s *Server) deleteCheckup(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	id := c.Param("id")
	ctx := c.Request.Context()
	res, err := s.Pool.Exec(ctx, `DELETE FROM checkups WHERE id=$1 AND user_id=$2`, id, userID)
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

func (s *Server) uploadReports(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	checkupID := c.Param("id")
	ctx := c.Request.Context()
	var n int
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM checkup_reports WHERE checkup_id=$1`, checkupID).Scan(&n); err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	if n >= 9 {
		resp.BadRequest(c, "最多上传9张图片", "E_PARAM_INVALID", nil)
		return
	}

	var own int
	if err := s.Pool.QueryRow(ctx, `SELECT 1 FROM checkups WHERE id=$1 AND user_id=$2`, checkupID, userID).Scan(&own); err != nil {
		resp.NotFound(c, "产检记录不存在")
		return
	}

	_ = c.Request.ParseMultipartForm(32 << 20)
	var files []*multipart.FileHeader
	if fh := c.Request.MultipartForm; fh != nil {
		if arr, ok := fh.File["images"]; ok {
			files = append(files, arr...)
		}
		if arr, ok := fh.File["images[]"]; ok {
			files = append(files, arr...)
		}
	}
	if len(files) == 0 {
		if f, err := c.FormFile("images"); err == nil {
			files = []*multipart.FileHeader{f}
		}
	}
	if len(files) == 0 {
		resp.BadRequest(c, "请选择图片", "E_PARAM_INVALID", nil)
		return
	}
	if n+len(files) > 9 {
		resp.BadRequest(c, "最多上传9张图片", "E_PARAM_INVALID", nil)
		return
	}

	summary := c.PostForm("summary")
	note := c.PostForm("note")
	outImgs := []gin.H{}
	for _, fh := range files {
		if fh.Size > 10<<20 {
			resp.BadRequest(c, "单张图片过大", "E_PARAM_INVALID", nil)
			return
		}
		ct := fh.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "image/") {
			resp.BadRequest(c, "仅支持图片格式", "E_PARAM_INVALID", nil)
			return
		}
		f, err := fh.Open()
		if err != nil {
			resp.Internal(c, "读取文件失败")
			return
		}
		key := fmt.Sprintf("%s/%s/%s%s", userID.String(), checkupID, uuid.New().String(), extFromCT(ct))
		url, err := s.Store.Put(ctx, key, f, fh.Size, ct)
		f.Close()
		if err != nil {
			resp.Internal(c, "上传存储失败")
			return
		}
		var rid uuid.UUID
		err = s.Pool.QueryRow(ctx, `
INSERT INTO checkup_reports (checkup_id, storage_key, public_url, thumb_url, sort_order)
VALUES ($1,$2,$3,$3,$4) RETURNING id`,
			checkupID, key, url, n+len(outImgs)).Scan(&rid)
		if err != nil {
			resp.Internal(c, "保存记录失败")
			return
		}
		outImgs = append(outImgs, gin.H{"id": rid.String(), "url": url, "thumbnail": url})
	}

	if summary != "" || note != "" {
		_, _ = s.Pool.Exec(ctx, `UPDATE checkups SET summary=COALESCE(NULLIF($3,''), summary), note=COALESCE(NULLIF($4,''), note), updated_at=now() WHERE id=$1 AND user_id=$2`,
			checkupID, userID, summary, note)
	}

	resp.OK(c, gin.H{"images": outImgs, "summary": summary})
}

func extFromCT(ct string) string {
	switch {
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "webp"):
		return ".webp"
	default:
		return ".jpg"
	}
}

