package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sifan077/PowerURL/internal/app/model"
	"github.com/sifan077/PowerURL/internal/app/repository"
)

type mockLinkRepository struct {
	createFn func(ctx context.Context, link *model.Link) error
	getFn    func(ctx context.Context, code string) (*model.Link, error)
	listFn   func(ctx context.Context, limit, offset int) ([]model.Link, error)
	updateFn func(ctx context.Context, link *model.Link) error
}

func (m *mockLinkRepository) Create(ctx context.Context, link *model.Link) error {
	if m.createFn != nil {
		return m.createFn(ctx, link)
	}
	return nil
}

func (m *mockLinkRepository) GetByCode(ctx context.Context, code string) (*model.Link, error) {
	if m.getFn != nil {
		return m.getFn(ctx, code)
	}
	return nil, repository.ErrLinkNotFound
}

func (m *mockLinkRepository) List(ctx context.Context, limit, offset int) ([]model.Link, error) {
	if m.listFn != nil {
		return m.listFn(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockLinkRepository) Update(ctx context.Context, link *model.Link) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, link)
	}
	return nil
}

func TestLinkService_CreateLink(t *testing.T) {
	repo := &mockLinkRepository{
		createFn: func(ctx context.Context, link *model.Link) error {
			if link.Code == "" {
				t.Fatal("expected code to be set")
			}
			return nil
		},
	}

	svc := NewLinkService(repo)
	_, err := svc.CreateLink(context.Background(), CreateLinkInput{
		Code: "abc123",
		URL:  "https://example.com",
		Mode: "",
	})
	if err != nil {
		t.Fatalf("CreateLink returned error: %v", err)
	}
}

func TestLinkService_GetLink_NotFound(t *testing.T) {
	repo := &mockLinkRepository{
		getFn: func(ctx context.Context, code string) (*model.Link, error) {
			return nil, repository.ErrLinkNotFound
		},
	}

	svc := NewLinkService(repo)
	_, err := svc.GetLink(context.Background(), "missing")
	if !errors.Is(err, repository.ErrLinkNotFound) {
		t.Fatalf("expected ErrLinkNotFound, got %v", err)
	}
}

func TestLinkService_ListLinks(t *testing.T) {
	repo := &mockLinkRepository{
		listFn: func(ctx context.Context, limit, offset int) ([]model.Link, error) {
			return []model.Link{{Code: "a"}, {Code: "b"}}, nil
		},
	}
	svc := NewLinkService(repo)

	list, err := svc.ListLinks(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("ListLinks error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 links, got %d", len(list))
	}
}

func TestLinkService_UpdateLink(t *testing.T) {
	expires := time.Now().Add(time.Hour)
	repo := &mockLinkRepository{
		getFn: func(ctx context.Context, code string) (*model.Link, error) {
			return &model.Link{Code: code}, nil
		},
		updateFn: func(ctx context.Context, link *model.Link) error {
			if link.URL != "https://new.example.com" {
				t.Fatalf("expected updated URL, got %s", link.URL)
			}
			if link.ExpiresAt == nil || !link.ExpiresAt.Equal(expires) {
				t.Fatalf("expected expiresAt to be set")
			}
			return nil
		},
	}

	svc := NewLinkService(repo)
	url := "https://new.example.com"
	_, err := svc.UpdateLink(context.Background(), "abc", UpdateLinkInput{
		URL:       &url,
		ExpiresAt: &expires,
	})
	if err != nil {
		t.Fatalf("UpdateLink error: %v", err)
	}
}
