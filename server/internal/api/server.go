package api

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"pregnancy-tracker/server/internal/config"
	"pregnancy-tracker/server/internal/middleware"
	"pregnancy-tracker/server/internal/storage"
)

type Server struct {
	Cfg   *config.Config
	Pool  *pgxpool.Pool
	Store *storage.Client
}

func uid(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(middleware.CtxUserID)
	if !ok {
		return uuid.Nil, false
	}
	s, _ := v.(string)
	id, err := uuid.Parse(s)
	return id, err == nil
}

func (s *Server) Register(r *gin.Engine, authMW gin.HandlerFunc) {
	v1 := r.Group("/api/v1")
	v1.POST("/auth/wechat", s.postAuthWechat)

	need := v1.Group("")
	need.Use(authMW)
	{
		need.GET("/user/info", s.getUserInfo)
		need.PUT("/user/info", s.putUserInfo)

		need.GET("/checkups", s.listCheckups)
		need.GET("/checkups/:id", s.getCheckup)
		need.POST("/checkups", s.createCheckup)
		need.PUT("/checkups/:id", s.updateCheckup)
		need.DELETE("/checkups/:id", s.deleteCheckup)
		need.POST("/checkups/:id/reports", s.uploadReports)

		need.GET("/weights", s.listWeights)
		need.POST("/weights", s.createWeight)
		need.DELETE("/weights/:id", s.deleteWeight)

		need.POST("/fetal-movements/sessions", s.fmCreateSession)
		need.GET("/fetal-movements/sessions", s.fmListSessions)
		need.POST("/fetal-movements/sessions/:id/events", s.fmEvent)
		need.POST("/fetal-movements/sessions/:id/finish", s.fmFinish)
		need.GET("/fetal-movements/summary", s.fmSummary)

		need.GET("/contractions", s.listContractions)
		need.POST("/contractions", s.createContraction)

		need.GET("/checklist", s.getChecklist)
		need.PUT("/checklist/items/:id", s.putChecklistItem)
		need.POST("/checklist/items", s.postChecklistItem)
		need.POST("/checklist/reset", s.resetChecklist)

		need.GET("/pregnancy/weeks/:week", s.getPregnancyWeek)
		need.PUT("/pregnancy/tasks/:taskId", s.putPregnancyTask)

		need.GET("/articles", s.listArticles)
		need.GET("/articles/:id", s.getArticle)
		need.POST("/favorites", s.postFavorite)
		need.DELETE("/favorites", s.deleteFavorite)

		need.GET("/settings", s.getSettings)
		need.PUT("/settings", s.putSettings)

		need.POST("/exports", s.postExport)
		need.GET("/exports/:exportId", s.getExport)
		need.GET("/exports/:exportId/download", s.downloadExport)

		need.POST("/ai/chat", s.postAIChat)
	}
}
