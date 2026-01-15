package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sifan077/PowerURL/internal/app/model"
	"github.com/sifan077/PowerURL/internal/app/repository"
	"github.com/sifan077/PowerURL/internal/app/service"
	httpUtil "github.com/sifan077/PowerURL/internal/http/util"
	"github.com/sifan077/PowerURL/internal/http/view"
	"go.uber.org/zap"
)

const tokenTTL = 60 * time.Second

// RedirectDeps groups dependencies required by redirect handlers.
type RedirectDeps struct {
	Logger         *zap.Logger
	Links          repository.LinkRepository
	Secret         []byte
	ClickPublisher *service.ClickPublisher
}

// RedirectHandler implements the redirect + intermediate flows.
type RedirectHandler struct {
	logger         *zap.Logger
	links          repository.LinkRepository
	tokens         *httpUtil.TokenSigner
	clickPublisher *service.ClickPublisher
}

// NewRedirectHandler creates a redirect handler with the provided dependencies.
func NewRedirectHandler(deps RedirectDeps) *RedirectHandler {
	logger := deps.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RedirectHandler{
		logger:         logger,
		links:          deps.Links,
		tokens:         httpUtil.NewTokenSigner(deps.Secret, tokenTTL),
		clickPublisher: deps.ClickPublisher,
	}
}

// Register wires redirect routes onto the provided router.
func (h *RedirectHandler) Register(router fiber.Router) {
	router.Get("/", h.Health)
	router.Get("/health", h.Health)
	router.Get("/:code", h.Resolve)
	router.Get("/:code/_go/:token", h.Go)
}

// Health is a simple root endpoint so we know the service is running.
func (h *RedirectHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"service": "PowerURL",
		"status":  "ok",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// Resolve handles GET /:code and decides between direct jump and intermediate page.
func (h *RedirectHandler) Resolve(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "missing link code",
		})
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	link, loadErr := h.loadLink(ctx, code)
	if loadErr != nil {
		return c.Status(loadErr.StatusCode).JSON(fiber.Map{
			"error": loadErr.Message,
		})
	}

	switch link.Mode {
	case "", "direct":
		// Publish click event for direct mode
		if h.clickPublisher != nil {
			go h.publishClickEvent(code, c)
		}
		h.logger.Debug("redirecting short link", zap.String("code", code), zap.String("target", link.URL))
		return c.Redirect(link.URL, fiber.StatusFound)
	case "click", "timer":
		return h.renderIntermediate(c, link)
	default:
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error": "redirect mode is not supported",
			"mode":  link.Mode,
		})
	}
}

// Go verifies the provided token and issues the final redirect.
func (h *RedirectHandler) Go(c *fiber.Ctx) error {
	code := c.Params("code")
	token := c.Params("token")
	if code == "" || token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "missing code or token",
		})
	}

	if err := h.tokens.Validate(code, token); err != nil {
		if errors.Is(err, httpUtil.ErrInvalidToken) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		h.logger.Error("failed to validate redirect token", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to validate token",
		})
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	link, loadErr := h.loadLink(ctx, code)
	if loadErr != nil {
		return c.Status(loadErr.StatusCode).JSON(fiber.Map{
			"error": loadErr.Message,
		})
	}

	// Publish click event asynchronously
	if h.clickPublisher != nil {
		go h.publishClickEvent(code, c)
	}

	h.logger.Debug("final redirect", zap.String("code", code), zap.String("target", link.URL))
	return c.Redirect(link.URL, fiber.StatusFound)
}

func (h *RedirectHandler) renderIntermediate(c *fiber.Ctx, link *model.Link) error {
	token, err := h.tokens.Issue(link.Code)
	if err != nil {
		h.logger.Error("failed to issue redirect token", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to prepare redirect",
		})
	}

	continueURL := fmt.Sprintf("/%s/_go/%s", link.Code, token)
	html, err := view.RenderRedirectPage(view.RedirectPageData{
		Title:        "Continue to destination",
		Code:         link.Code,
		TargetURL:    link.URL,
		ContinueURL:  continueURL,
		Mode:         link.Mode,
		TimerSeconds: link.TimerSeconds,
		Token:        token,
	})
	if err != nil {
		h.logger.Error("failed to render redirect page", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to render page",
		})
	}

	return c.
		Type("html", "utf-8").
		SendString(html)
}

type linkLoadError struct {
	StatusCode int
	Message    string
}

func (h *RedirectHandler) loadLink(ctx context.Context, code string) (*model.Link, *linkLoadError) {
	link, err := h.links.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrLinkNotFound) {
			return nil, &linkLoadError{
				StatusCode: fiber.StatusNotFound,
				Message:    "short link not found",
			}
		}
		h.logger.Error("failed to load link", zap.Error(err), zap.String("code", code))
		return nil, &linkLoadError{
			StatusCode: fiber.StatusInternalServerError,
			Message:    "internal server error",
		}
	}

	if link.Disabled {
		return nil, &linkLoadError{
			StatusCode: fiber.StatusGone,
			Message:    "link is disabled",
		}
	}
	if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
		return nil, &linkLoadError{
			StatusCode: fiber.StatusGone,
			Message:    "link expired",
		}
	}

	return link, nil
}

func (h *RedirectHandler) publishClickEvent(code string, c *fiber.Ctx) {
	if err := h.clickPublisher.Publish(code, c.IP(), c.Get("User-Agent")); err != nil {
		h.logger.Error("failed to publish click event", zap.Error(err), zap.String("code", code))
	}
}
