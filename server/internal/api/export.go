package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"pregnancy-tracker/server/pkg/resp"
)

func (s *Server) postExport(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	var body struct {
		Types  []string `json:"types"`
		Format string   `json:"format"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Types) == 0 {
		resp.BadRequest(c, "请选择导出类型", "E_PARAM_INVALID", nil)
		return
	}
	if body.Format == "" {
		body.Format = "csv"
	}
	ctx := c.Request.Context()
	since := time.Now().Add(-s.Cfg.ExportCooldown)
	var recent int
	_ = s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM export_jobs WHERE user_id=$1 AND created_at > $2`, userID, since).Scan(&recent)
	if recent > 0 {
		resp.TooManyRequests(c, "导出过于频繁，请稍后再试", "E_EXPORT_TOO_FREQUENT")
		return
	}

	var jobID uuid.UUID
	err := s.Pool.QueryRow(ctx, `INSERT INTO export_jobs (user_id, types, format, status) VALUES ($1,$2,$3,'processing') RETURNING id`,
		userID, body.Types, body.Format).Scan(&jobID)
	if err != nil {
		resp.Internal(c, "创建任务失败")
		return
	}

	go s.runExportJob(jobID, userID, body.Types, body.Format)

	resp.OK(c, gin.H{"exportId": jobID.String(), "status": "processing"})
}

func (s *Server) getExport(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	eid := c.Param("exportId")
	ctx := c.Request.Context()
	var status, format string
	var url, fpath sql.NullString
	err := s.Pool.QueryRow(ctx, `SELECT status, format, public_url, file_path FROM export_jobs WHERE id=$1 AND user_id=$2`, eid, userID).Scan(&status, &format, &url, &fpath)
	if err == pgx.ErrNoRows {
		resp.NotFound(c, "任务不存在")
		return
	}
	if err != nil {
		resp.Internal(c, "查询失败")
		return
	}
	out := gin.H{"exportId": eid, "status": status}
	if url.Valid && url.String != "" {
		out["downloadUrl"] = url.String
	} else if status == "ready" && fpath.Valid {
		out["downloadUrl"] = s.Cfg.PublicBaseURL + "/api/v1/exports/" + eid + "/download"
	}
	resp.OK(c, out)
}

func (s *Server) downloadExport(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	eid := c.Param("exportId")
	ctx := c.Request.Context()
	var fpath, status string
	err := s.Pool.QueryRow(ctx, `SELECT file_path, status FROM export_jobs WHERE id=$1 AND user_id=$2`, eid, userID).Scan(&fpath, &status)
	if err != nil || status != "ready" || fpath == "" {
		resp.NotFound(c, "文件未就绪")
		return
	}
	c.File(fpath)
}

func (s *Server) runExportJob(jobID uuid.UUID, userID uuid.UUID, types []string, format string) {
	ctx := context.Background()
	dir := filepath.Join(filepath.Dir(s.Cfg.LocalUploadDir), "exports")
	_ = os.MkdirAll(dir, 0o755)
	name := fmt.Sprintf("%s.%s", jobID.String(), format)
	full := filepath.Join(dir, name)

	var buf bytes.Buffer
	if format == "json" {
		data := s.collectExportData(ctx, userID, types)
		b, _ := json.MarshalIndent(data, "", "  ")
		buf.Write(b)
	} else {
		w := csv.NewWriter(&buf)
		_ = w.Write([]string{"section", "data"})
		for _, t := range types {
			rows := s.exportSection(ctx, userID, t)
			for _, r := range rows {
				_ = w.Write(r)
			}
		}
		w.Flush()
	}
	if err := os.WriteFile(full, buf.Bytes(), 0o644); err != nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE export_jobs SET status='failed', error_message=$2, completed_at=now() WHERE id=$1`, jobID, err.Error())
		return
	}
	publicURL := s.Cfg.PublicBaseURL + "/api/v1/exports/" + jobID.String() + "/download"
	_, _ = s.Pool.Exec(ctx, `UPDATE export_jobs SET status='ready', file_path=$2, public_url=$3, completed_at=now() WHERE id=$1`, jobID, full, publicURL)
}

func (s *Server) collectExportData(ctx context.Context, userID uuid.UUID, types []string) map[string]interface{} {
	out := map[string]interface{}{}
	for _, t := range types {
		out[t] = s.exportSection(ctx, userID, t)
	}
	return out
}

func (s *Server) exportSection(ctx context.Context, userID uuid.UUID, section string) [][]string {
	switch section {
	case "weights":
		rows, _ := s.Pool.Query(ctx, `SELECT recorded_at, weight::text, note FROM weights WHERE user_id=$1 ORDER BY recorded_at`, userID)
		if rows == nil {
			return nil
		}
		defer rows.Close()
		var res [][]string
		for rows.Next() {
			var d time.Time
			var w, n string
			if rows.Scan(&d, &w, &n) == nil {
				res = append(res, []string{"weight", d.Format("2006-01-02"), w, n})
			}
		}
		return res
	case "checkups":
		rows, _ := s.Pool.Query(ctx, `SELECT checkup_date, checkup_type, hospital, note FROM checkups WHERE user_id=$1`, userID)
		if rows == nil {
			return nil
		}
		defer rows.Close()
		var res [][]string
		for rows.Next() {
			var cd time.Time
			var typ, h, n string
			if rows.Scan(&cd, &typ, &h, &n) == nil {
				res = append(res, []string{"checkup", cd.Format("2006-01-02"), typ, h, n})
			}
		}
		return res
	case "fetalMovements":
		rows, _ := s.Pool.Query(ctx, `SELECT started_at, ended_at, count::text FROM fm_sessions WHERE user_id=$1 AND status='finished'`, userID)
		if rows == nil {
			return nil
		}
		defer rows.Close()
		var res [][]string
		for rows.Next() {
			var st, en sql.NullTime
			var cnt string
			if rows.Scan(&st, &en, &cnt) != nil {
				continue
			}
			sa, ea := "", ""
			if st.Valid {
				sa = st.Time.UTC().Format(time.RFC3339)
			}
			if en.Valid {
				ea = en.Time.UTC().Format(time.RFC3339)
			}
			res = append(res, []string{"fm", sa, ea, cnt})
		}
		return res
	case "contractions":
		rows, _ := s.Pool.Query(ctx, `SELECT started_at, ended_at, duration_sec::text FROM contractions WHERE user_id=$1`, userID)
		if rows == nil {
			return nil
		}
		defer rows.Close()
		var res [][]string
		for rows.Next() {
			var st, en time.Time
			var ds string
			if rows.Scan(&st, &en, &ds) == nil {
				res = append(res, []string{"contraction", st.Format(time.RFC3339), en.Format(time.RFC3339), ds})
			}
		}
		return res
	case "checklist":
		rows, _ := s.Pool.Query(ctx, `SELECT title, checked::text, note FROM checklist_items WHERE user_id=$1`, userID)
		if rows == nil {
			return nil
		}
		defer rows.Close()
		var res [][]string
		for rows.Next() {
			var t, ch, n string
			if rows.Scan(&t, &ch, &n) == nil {
				res = append(res, []string{"checklist", t, ch, n})
			}
		}
		return res
	default:
		return nil
	}
}
