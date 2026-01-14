package server

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/sifan077/PowerURL/internal/app/repository"
	inthttp "github.com/sifan077/PowerURL/internal/http/handler"
	"go.uber.org/zap"
)

// Dependencies bundles infrastructure dependencies required by the HTTP server.
type Dependencies struct {
	Logger    *zap.Logger
	Postgres  *pgxpool.Pool
	Redis     *redis.Client
	NATS      *nats.Conn
	JetStream nats.JetStreamContext
	Links     repository.LinkRepository
	Secret    []byte
}

// Server wraps the Fiber application and its dependencies.
type Server struct {
	app  *fiber.App
	deps Dependencies
}

// New creates a new HTTP server instance with default routes.
func New(deps Dependencies) *Server {
	app := fiber.New()

	s := &Server{
		app:  app,
		deps: deps,
	}

	s.registerRoutes()
	return s
}

// Listen starts the Fiber server on the given address.
func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

// Shutdown gracefully stops the Fiber server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}

func (s *Server) registerRoutes() {
	redirectHandler := inthttp.NewRedirectHandler(inthttp.RedirectDeps{
		Logger: s.deps.Logger,
		Links:  s.deps.Links,
		Secret: s.deps.Secret,
	})
	redirectHandler.Register(s.app)
}
