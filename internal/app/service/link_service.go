package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sifan077/PowerURL/internal/app/model"
	"github.com/sifan077/PowerURL/internal/app/repository"
)

// LinkService defines behaviour-level operations on links.
type LinkService interface {
	CreateLink(ctx context.Context, input CreateLinkInput) (*model.Link, error)
	GetLink(ctx context.Context, code string) (*model.Link, error)
	ListLinks(ctx context.Context, limit, offset int) ([]model.Link, error)
	UpdateLink(ctx context.Context, code string, input UpdateLinkInput) (*model.Link, error)
}

type linkService struct {
	repo repository.LinkRepository
}

// NewLinkService returns a service implementation backed by the given repository.
func NewLinkService(repo repository.LinkRepository) LinkService {
	return &linkService{repo: repo}
}

// CreateLinkInput captures data required to create a link.
type CreateLinkInput struct {
	Code         string
	URL          string
	Mode         string
	TimerSeconds int
	Disabled     bool
	ExpiresAt    *time.Time
}

// UpdateLinkInput captures fields that can be changed on an existing link.
type UpdateLinkInput struct {
	URL          *string
	Mode         *string
	TimerSeconds *int
	Disabled     *bool
	ExpiresAt    *time.Time
}

func (s *linkService) CreateLink(ctx context.Context, input CreateLinkInput) (*model.Link, error) {
	link := &model.Link{
		Code:         input.Code,
		URL:          input.URL,
		Mode:         input.Mode,
		TimerSeconds: input.TimerSeconds,
		Disabled:     input.Disabled,
		ExpiresAt:    input.ExpiresAt,
	}

	if link.Mode == "" {
		link.Mode = "direct"
	}

	if err := s.repo.Create(ctx, link); err != nil {
		return nil, fmt.Errorf("create link: %w", err)
	}
	return link, nil
}

func (s *linkService) GetLink(ctx context.Context, code string) (*model.Link, error) {
	link, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("get link: %w", err)
	}
	return link, nil
}

func (s *linkService) ListLinks(ctx context.Context, limit, offset int) ([]model.Link, error) {
	links, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	return links, nil
}

func (s *linkService) UpdateLink(ctx context.Context, code string, input UpdateLinkInput) (*model.Link, error) {
	link, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("load link: %w", err)
	}

	if input.URL != nil {
		link.URL = *input.URL
	}
	if input.Mode != nil {
		link.Mode = *input.Mode
	}
	if input.TimerSeconds != nil {
		link.TimerSeconds = *input.TimerSeconds
	}
	if input.Disabled != nil {
		link.Disabled = *input.Disabled
	}
	if input.ExpiresAt != nil {
		link.ExpiresAt = input.ExpiresAt
	}

	if err := s.repo.Update(ctx, link); err != nil {
		return nil, fmt.Errorf("update link: %w", err)
	}
	return link, nil
}
