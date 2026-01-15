package server

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/sifan077/PowerURL/internal/app/repository"
	"github.com/sifan077/PowerURL/internal/app/service"
	inthttp "github.com/sifan077/PowerURL/internal/http/handler"
	"github.com/sifan077/PowerURL/internal/http/middleware"
	"go.uber.org/zap"
)

// Dependencies bundles infrastructure dependencies required by the HTTP server.
type Dependencies struct {
	Logger     *zap.Logger
	Postgres   *pgxpool.Pool
	Redis      *redis.Client
	NATS       *nats.Conn
	JetStream  nats.JetStreamContext
	Links      repository.LinkRepository
	ClickEvents repository.ClickEventRepository
	Secret     []byte
}

// Server wraps the Fiber application and its dependencies.
type Server struct {
	app                 *fiber.App
	deps                Dependencies
	clickTimeoutChecker *service.ClickTimeoutChecker
}

// New creates a new HTTP server instance with default routes.
func New(deps Dependencies) *Server {
	app := fiber.New()

	s := &Server{
		app:  app,
		deps: deps,
	}

	s.registerMiddleware()
	s.registerRoutes()
	s.startBackgroundServices()
	return s
}

// Listen starts the Fiber server on the given address.
func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

// Shutdown gracefully stops the Fiber server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.clickTimeoutChecker != nil {
		s.clickTimeoutChecker.Stop()
	}
	return s.app.ShutdownWithContext(ctx)
}

func (s *Server) startBackgroundServices() {
	// Start click timeout checker with 60 seconds TTL
	s.clickTimeoutChecker = service.NewClickTimeoutChecker(s.deps.Logger, s.deps.ClickEvents, 60*time.Second)
	s.clickTimeoutChecker.Start()
}

func (s *Server) registerMiddleware() {
	rateLimitConfig := middleware.DefaultRateLimitConfig()

	s.app.Use(middleware.Recovery(s.deps.Logger))
	s.app.Use(middleware.RequestID())
	s.app.Use(middleware.Logger(s.deps.Logger))
	s.app.Use(middleware.CORS())
	s.app.Use(middleware.RateLimit(s.deps.Redis, rateLimitConfig, s.deps.Logger))
}

func (s *Server) registerRoutes() {
	clickPublisher := service.NewClickPublisher(s.deps.JetStream)
	clickConsumer := service.NewClickConsumer(s.deps.JetStream, s.deps.Logger, s.deps.ClickEvents)

	// Start click event consumer
	if err := clickConsumer.Start(); err != nil {
		s.deps.Logger.Error("failed to start click consumer", zap.Error(err))
	}

	redirectHandler := inthttp.NewRedirectHandler(inthttp.RedirectDeps{
		Logger:         s.deps.Logger,
		Links:          s.deps.Links,
		ClickEvents:    s.deps.ClickEvents,
		Secret:         s.deps.Secret,
		ClickPublisher: clickPublisher,
	})
	redirectHandler.Register(s.app)
}
