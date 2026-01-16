package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sifan077/PowerURL/internal/app/service"
	"go.uber.org/zap"
)

// APIDeps groups dependencies required by API handlers.
type APIDeps struct {
	Logger      *zap.Logger
	LinkService service.LinkService
}

// APIHandler implements the management API endpoints.
type APIHandler struct {
	logger      *zap.Logger
	linkService service.LinkService
}

// NewAPIHandler creates an API handler with the provided dependencies.
func NewAPIHandler(deps APIDeps) *APIHandler {
	logger := deps.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &APIHandler{
		logger:      logger,
		linkService: deps.LinkService,
	}
}

// Register wires API routes onto the provided router.
func (h *APIHandler) Register(router fiber.Router) {
	api := router.Group("/api")
	{
		links := api.Group("/links")
		{
			links.Post("/", h.CreateLink)
			links.Get("/", h.ListLinks)
			links.Get("/:code", h.GetLink)
			links.Patch("/:code", h.UpdateLink)
		}
	}
}

// CreateLinkRequest represents the request body for creating a link.
type CreateLinkRequest struct {
	Code         string     `json:"code,omitempty"`
	URL          string     `json:"url" validate:"required,url"`
	Mode         string     `json:"mode,omitempty" validate:"omitempty,oneof=direct click timer"`
	TimerSeconds int        `json:"timer_seconds,omitempty" validate:"omitempty,min=0,max=300"`
	Disabled     bool       `json:"disabled,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// CreateLinkResponse represents the response for creating a link.
type CreateLinkResponse struct {
	Code         string     `json:"code"`
	URL          string     `json:"url"`
	Mode         string     `json:"mode"`
	TimerSeconds int        `json:"timer_seconds"`
	Disabled     bool       `json:"disabled"`
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// CreateLink handles POST /api/links
func (h *APIHandler) CreateLink(c *fiber.Ctx) error {
	var req CreateLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "url is required",
		})
	}

	if req.Mode != "" && req.Mode != "direct" && req.Mode != "click" && req.Mode != "timer" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "mode must be one of: direct, click, timer",
		})
	}

	if req.TimerSeconds < 0 || req.TimerSeconds > 300 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "timer_seconds must be between 0 and 300",
		})
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	input := service.CreateLinkInput{
		Code:         req.Code,
		URL:          req.URL,
		Mode:         req.Mode,
		TimerSeconds: req.TimerSeconds,
		Disabled:     req.Disabled,
		ExpiresAt:    req.ExpiresAt,
	}

	link, err := h.linkService.CreateLink(ctx, input)
	if err != nil {
		h.logger.Error("failed to create link", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create link",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(CreateLinkResponse{
		Code:         link.Code,
		URL:          link.URL,
		Mode:         link.Mode,
		TimerSeconds: link.TimerSeconds,
		Disabled:     link.Disabled,
		ExpiresAt:    link.ExpiresAt,
		CreatedAt:    link.CreatedAt,
	})
}

// ListLinks handles GET /api/links
func (h *APIHandler) ListLinks(c *fiber.Ctx) error {
	limit := 20
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed := c.QueryInt("limit"); parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsed := c.QueryInt("offset"); parsed >= 0 {
			offset = parsed
		}
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	links, err := h.linkService.ListLinks(ctx, limit, offset)
	if err != nil {
		h.logger.Error("failed to list links", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list links",
		})
	}

	response := make([]CreateLinkResponse, len(links))
	for i, link := range links {
		response[i] = CreateLinkResponse{
			Code:         link.Code,
			URL:          link.URL,
			Mode:         link.Mode,
			TimerSeconds: link.TimerSeconds,
			Disabled:     link.Disabled,
			ExpiresAt:    link.ExpiresAt,
			CreatedAt:    link.CreatedAt,
		}
	}

	return c.JSON(fiber.Map{
		"links":  response,
		"limit":  limit,
		"offset": offset,
		"count":  len(response),
	})
}

// GetLink handles GET /api/links/:code
func (h *APIHandler) GetLink(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "code is required",
		})
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	link, err := h.linkService.GetLink(ctx, code)
	if err != nil {
		h.logger.Error("failed to get link", zap.Error(err), zap.String("code", code))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "link not found",
		})
	}

	return c.JSON(CreateLinkResponse{
		Code:         link.Code,
		URL:          link.URL,
		Mode:         link.Mode,
		TimerSeconds: link.TimerSeconds,
		Disabled:     link.Disabled,
		ExpiresAt:    link.ExpiresAt,
		CreatedAt:    link.CreatedAt,
	})
}

// UpdateLinkRequest represents the request body for updating a link.
type UpdateLinkRequest struct {
	URL          *string    `json:"url,omitempty" validate:"omitempty,url"`
	Mode         *string    `json:"mode,omitempty" validate:"omitempty,oneof=direct click timer"`
	TimerSeconds *int       `json:"timer_seconds,omitempty" validate:"omitempty,min=0,max=300"`
	Disabled     *bool      `json:"disabled,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// UpdateLink handles PATCH /api/links/:code
func (h *APIHandler) UpdateLink(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "code is required",
		})
	}

	var req UpdateLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Mode != nil && *req.Mode != "" && *req.Mode != "direct" && *req.Mode != "click" && *req.Mode != "timer" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "mode must be one of: direct, click, timer",
		})
	}

	if req.TimerSeconds != nil && (*req.TimerSeconds < 0 || *req.TimerSeconds > 300) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "timer_seconds must be between 0 and 300",
		})
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	input := service.UpdateLinkInput{
		URL:          req.URL,
		Mode:         req.Mode,
		TimerSeconds: req.TimerSeconds,
		Disabled:     req.Disabled,
		ExpiresAt:    req.ExpiresAt,
	}

	link, err := h.linkService.UpdateLink(ctx, code, input)
	if err != nil {
		h.logger.Error("failed to update link", zap.Error(err), zap.String("code", code))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "link not found",
		})
	}

	return c.JSON(CreateLinkResponse{
		Code:         link.Code,
		URL:          link.URL,
		Mode:         link.Mode,
		TimerSeconds: link.TimerSeconds,
		Disabled:     link.Disabled,
		ExpiresAt:    link.ExpiresAt,
		CreatedAt:    link.CreatedAt,
	})
}