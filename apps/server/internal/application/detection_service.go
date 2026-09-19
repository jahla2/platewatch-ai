package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
	"github.com/jahla2/platewatch-ai/apps/server/internal/ports"
)

type CreateDetectionInput struct {
	CameraID     string  `json:"camera_id"`
	TrackID      int64   `json:"track_id"`
	Plate        string  `json:"plate"`
	Confidence   float64 `json:"confidence"`
	SnapshotURL  string  `json:"snapshot_url"`
	PlateCropURL string  `json:"plate_crop_url"`
}

type DetectionService struct {
	repository ports.DetectionRepository
	watchlist  ports.Watchlist
	publisher  ports.DetectionPublisher
	now        func() time.Time
	newID      func() (string, error)
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
		newID:      randomID,
	}
}

func (s *DetectionService) Create(
	ctx context.Context,
	input CreateDetectionInput,
) (domain.DetectionEvent, error) {
	plate := domain.NormalizePlate(input.Plate)
	cameraID := strings.TrimSpace(input.CameraID)
	snapshotURL := strings.TrimSpace(input.SnapshotURL)
	plateCropURL := strings.TrimSpace(input.PlateCropURL)

	if cameraID == "" {
		return domain.DetectionEvent{}, errors.New("camera_id is required")
	}
	if input.TrackID <= 0 {
		return domain.DetectionEvent{}, errors.New("track_id must be positive")
	}
	if plate == "" {
		return domain.DetectionEvent{}, errors.New("plate is required")
	}
	if input.Confidence < 0 || input.Confidence > 1 {
		return domain.DetectionEvent{}, errors.New("confidence must be between 0 and 1")
	}
	if !validEvidenceURL(snapshotURL) || !validEvidenceURL(plateCropURL) {
		return domain.DetectionEvent{}, errors.New("evidence URLs must be empty or start with /evidence/")
	}

	flagged, err := s.watchlist.IsFlagged(ctx, plate)
	if err != nil {
		return domain.DetectionEvent{}, err
	}
	id, err := s.newID()
	if err != nil {
		return domain.DetectionEvent{}, err
	}

	event := domain.DetectionEvent{
		ID:           id,
		CameraID:     cameraID,
		TrackID:      input.TrackID,
		Plate:        plate,
		Confidence:   input.Confidence,
		Flagged:      flagged,
		SnapshotURL:  snapshotURL,
		PlateCropURL: plateCropURL,
		DetectedAt:   s.now().UTC(),
	}

	if err := s.repository.Save(ctx, event); err != nil {
		return domain.DetectionEvent{}, err
	}
	if err := s.publisher.Publish(ctx, event); err != nil {
		return domain.DetectionEvent{}, err
	}
	return event, nil
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

func randomID() (string, error) {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "evt_" + hex.EncodeToString(bytes[:]), nil
}
