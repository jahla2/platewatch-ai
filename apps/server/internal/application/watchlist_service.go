package application

import (
	"context"
	"errors"
	"strings"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
	"github.com/jahla2/platewatch-ai/apps/server/internal/ports"
)

type UpsertWatchlistInput struct {
	Plate  string `json:"plate"`
	Reason string `json:"reason"`
	Active bool   `json:"active"`
}

type WatchlistService struct {
	store ports.WatchlistStore
}

func NewWatchlistService(store ports.WatchlistStore) *WatchlistService {
	return &WatchlistService{store: store}
}

func (s *WatchlistService) List(ctx context.Context, limit int) ([]domain.WatchlistEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	return s.store.ListWatchlist(ctx, limit)
}

func (s *WatchlistService) Upsert(
	ctx context.Context,
	input UpsertWatchlistInput,
) (domain.WatchlistEntry, error) {
	plate := domain.NormalizePlate(input.Plate)
	if plate == "" {
		return domain.WatchlistEntry{}, errors.New("plate is required")
	}

	reason := strings.TrimSpace(input.Reason)
	if len(reason) > 500 {
		return domain.WatchlistEntry{}, errors.New("reason must be 500 characters or fewer")
	}

	return s.store.UpsertWatchlist(ctx, domain.WatchlistEntry{
		Plate:  plate,
		Reason: reason,
		Active: input.Active,
	})
}

func (s *WatchlistService) Delete(ctx context.Context, plate string) error {
	normalized := domain.NormalizePlate(plate)
	if normalized == "" {
		return errors.New("plate is required")
	}
	return s.store.DeleteWatchlist(ctx, normalized)
}
