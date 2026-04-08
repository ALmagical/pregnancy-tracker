package main

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"pregnancy-tracker/server/internal/api"
	"pregnancy-tracker/server/internal/config"
	"pregnancy-tracker/server/internal/db"
	"pregnancy-tracker/server/internal/middleware"
	"pregnancy-tracker/server/internal/storage"
)

func main() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("server/.env")

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	store, err := storage.New(cfg)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	absUpload, _ := filepath.Abs(cfg.LocalUploadDir)
	r.Static("/files", absUpload)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	srv := &api.Server{Cfg: cfg, Pool: pool, Store: store}
	srv.Register(r, middleware.JWT(cfg))

	addr := cfg.HTTPAddr
	log.Printf("listening %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
