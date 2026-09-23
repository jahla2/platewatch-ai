package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
	"github.com/jahla2/platewatch-ai/apps/server/internal/ports"
)

type CreateDetectionInput struct {
	IdempotencyKey string  `json:"-"`
	CameraID       string  `json:"camera_id"`
	TrackID        int64   `json:"track_id"`
	PlateText      string  `json:"plate_text"`
	PlateKey       string  `json:"plate_key,omitempty"`
	Confidence     float64 `json:"confidence"`
	SnapshotURL    string  `json:"snapshot_url"`
	PlateCropURL   string  `json:"plate_crop_url"`
}

type DetectionService struct {
	repository ports.DetectionRepository
	watchlist  ports.Watchlist
	publisher  ports.DetectionPublisher
	now        func() time.Time
}

func NewDetectionService(
	repository ports.DetectionRepository,
	watchlist ports.Watchlist,
	publisher ports.DetectionPublisher,
) *DetectionService {
	return &DetectionService{
		repository: repository,
		watchlist:  watchlist,
		publisher:  publisher,
		now:        time.Now,
	}
}

func (s *DetectionService) Create(
	ctx context.Context,
	input CreateDetectionInput,
) (domain.DetectionEvent, bool, error) {
	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	plateText := strings.TrimSpace(input.PlateText)
	plateKey := domain.CanonicalizePlateText(plateText)
	cameraID := strings.TrimSpace(input.CameraID)
	snapshotURL := strings.TrimSpace(input.SnapshotURL)
	plateCropURL := strings.TrimSpace(input.PlateCropURL)

	if idempotencyKey == "" {
		return domain.DetectionEvent{}, false, errors.New("idempotency key is required")
	}
	if len(idempotencyKey) > 128 {
		return domain.DetectionEvent{}, false, errors.New("idempotency key is too long")
	}
	if cameraID == "" {
		return domain.DetectionEvent{}, false, errors.New("camera_id is required")
	}
	if input.TrackID <= 0 {
		return domain.DetectionEvent{}, false, errors.New("track_id must be positive")
	}
	if plateText == "" || plateKey == "" {
		return domain.DetectionEvent{}, false, errors.New("plate_text is required")
	}
	if providedKey := strings.TrimSpace(input.PlateKey); providedKey != "" && providedKey != plateKey {
		return domain.DetectionEvent{}, false, errors.New("plate_key does not match plate_text")
	}
	if input.Confidence < 0 || input.Confidence > 1 {
		return domain.DetectionEvent{}, false, errors.New("confidence must be between 0 and 1")
	}
	if !validEvidenceURL(snapshotURL) || !validEvidenceURL(plateCropURL) {
		return domain.DetectionEvent{}, false, errors.New("evidence URLs must be empty or start with /evidence/")
	}

	flagged, err := s.watchlist.IsFlagged(ctx, plateKey)
	if err != nil {
		return domain.DetectionEvent{}, false, err
	}

	event := domain.DetectionEvent{
		ID:             deterministicEventID(idempotencyKey),
		IdempotencyKey: idempotencyKey,
		CameraID:       cameraID,
		TrackID:        input.TrackID,
		PlateText:      plateText,
		PlateKey:       plateKey,
		Confidence:     input.Confidence,
		Flagged:        flagged,
		SnapshotURL:    snapshotURL,
		PlateCropURL:   plateCropURL,
		DetectedAt:     s.now().UTC(),
	}

	created, err := s.repository.SaveIfAbsent(ctx, event)
	if err != nil {
		return domain.DetectionEvent{}, false, err
	}
	if !created {
		return event, false, nil
	}
	if err := s.publisher.Publish(ctx, event); err != nil {
		return domain.DetectionEvent{}, false, err
	}
	return event, true, nil
}

func (s *DetectionService) List(ctx context.Context, limit int) ([]domain.DetectionEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repository.List(ctx, limit)
}

func validEvidenceURL(value string) bool {
	return value == "" || strings.HasPrefix(value, "/evidence/")
}

func deterministicEventID(idempotencyKey string) string {
	sum := sha256.Sum256([]byte(idempotencyKey))
	return "evt_" + hex.EncodeToString(sum[:12])
}
