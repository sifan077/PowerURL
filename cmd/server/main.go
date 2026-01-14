package main

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/sifan077/PowerURL/config"
	appmodel "github.com/sifan077/PowerURL/internal/app/model"
	apprepository "github.com/sifan077/PowerURL/internal/app/repository"
	appserver "github.com/sifan077/PowerURL/internal/app/server"
	"github.com/sifan077/PowerURL/internal/infra/logger"
	infraNATS "github.com/sifan077/PowerURL/internal/infra/nats"
	infraPostgres "github.com/sifan077/PowerURL/internal/infra/postgres"
	infraPrometheus "github.com/sifan077/PowerURL/internal/infra/prometheus"
	infraRedis "github.com/sifan077/PowerURL/internal/infra/redis"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	isDev := os.Getenv("APP_ENV") != "production"
	log := logger.MustInit(logger.Config{
		Development: isDev,
		Level:       os.Getenv("LOG_LEVEL"),
	})
	defer func() { _ = logger.Sync() }()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config", zap.Error(err))
	}

	log.Info("Configuration loaded successfully",
		zap.String("postgres_user", cfg.Postgres.User),
		zap.String("postgres_host", cfg.Postgres.Host),
		zap.Int("postgres_port", cfg.Postgres.Port),
		zap.String("postgres_db", cfg.Postgres.Database),
		zap.String("redis_host", cfg.Redis.Host),
		zap.Int("redis_port", cfg.Redis.Port),
		zap.String("nats_host", cfg.NATS.Host),
		zap.Int("nats_port", cfg.NATS.Port),
		zap.Int("nats_monitor_port", cfg.NATS.MonitorPort),
	)

	gormDB, err := infraPostgres.NewGorm(cfg.Postgres)
	if err != nil {
		log.Fatal("Failed to open GORM connection", zap.Error(err))
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatal("Failed to access underlying SQL DB", zap.Error(err))
	}
	defer sqlDB.Close()

	if err := infraPostgres.AutoMigrate(ctx, gormDB, &appmodel.Link{}); err != nil {
		log.Fatal("Failed to run database migrations", zap.Error(err))
	}

	pool, err := infraPostgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		log.Fatal("Failed to connect to Postgres", zap.Error(err))
	}
	defer pool.Close()

	log.Info("Connected to Postgres successfully")

	redisClient, err := infraRedis.NewClient(ctx, cfg.Redis)
	if err != nil {
		log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()
	log.Info("Connected to Redis successfully")

	natsConn, js, err := infraNATS.Connect(cfg.NATS)
	if err != nil {
		log.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer natsConn.Drain()
	log.Info("Connected to NATS successfully", zap.Bool("jetstream_ready", js != nil))

	if !isDev {
		promServer := infraPrometheus.NewServer(cfg.Prometheus)
		go func() {
			log.Info("Starting Prometheus metrics server",
				zap.Int("port", cfg.Prometheus.Port))
			if err := promServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("Prometheus metrics server stopped unexpectedly", zap.Error(err))
			}
		}()
		defer func() {
			if err := promServer.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Warn("Failed to close Prometheus server", zap.Error(err))
			}
		}()
	} else {
		log.Info("Skipping Prometheus metrics server in development mode")
	}

	linkRepo := apprepository.NewLinkRepository(gormDB)

	server := appserver.New(appserver.Dependencies{
		Logger:    log,
		Postgres:  pool,
		Redis:     redisClient,
		NATS:      natsConn,
		JetStream: js,
		Links:     linkRepo,
	})

	if err := server.Listen(":8080"); err != nil {
		log.Fatal("Fiber server exited", zap.Error(err))
	}
}
