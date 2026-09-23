package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
	"github.com/jahla2/platewatch-ai/apps/server/internal/ports"
)

type CreateDetectionInput struct {
	CameraID     string  `json:"camera_id"`
	TrackID      int64   `json:"track_id"`
	PlateText    string  `json:"plate_text"`
	PlateKey     string  `json:"plate_key,omitempty"`
	Confidence   float64 `json:"confidence"`
	SnapshotURL  string  `json:"snapshot_url"`
	PlateCropURL string  `json:"plate_crop_url"`
}

type DetectionPage struct {
	Items      []domain.DetectionEvent `json:"items"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

type detectionCursorPayload struct {
	DetectedAt time.Time `json:"detected_at"`
	ID         string    `json:"id"`
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
	idempotencyKey string,
) (event domain.DetectionEvent, created bool, err error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return domain.DetectionEvent{}, false, NewValidationError("Idempotency-Key is required")
	}
	if len(idempotencyKey) > 200 {
		return domain.DetectionEvent{}, false, NewValidationError(
			"Idempotency-Key must be 200 characters or fewer",
		)
	}

	plateText := strings.TrimSpace(input.PlateText)
	plateKey := domain.CanonicalizePlateText(plateText)
	cameraID := strings.TrimSpace(input.CameraID)
	snapshotURL := strings.TrimSpace(input.SnapshotURL)
	plateCropURL := strings.TrimSpace(input.PlateCropURL)

	if cameraID == "" {
		return domain.DetectionEvent{}, false, NewValidationError("camera_id is required")
	}
	if input.TrackID <= 0 {
		return domain.DetectionEvent{}, false, NewValidationError("track_id must be positive")
	}
	if plateText == "" || plateKey == "" {
		return domain.DetectionEvent{}, false, NewValidationError("plate_text is required")
	}
	if len(plateText) > 128 || len(plateKey) > 128 {
		return domain.DetectionEvent{}, false, NewValidationError(
			"plate text must be 128 characters or fewer",
		)
	}
	if providedKey := strings.TrimSpace(input.PlateKey); providedKey != "" && providedKey != plateKey {
		return domain.DetectionEvent{}, false, NewValidationError(
			"plate_key does not match plate_text",
		)
	}
	if input.Confidence < 0 || input.Confidence > 1 {
		return domain.DetectionEvent{}, false, NewValidationError(
			"confidence must be between 0 and 1",
		)
	}
	if !validEvidenceURL(snapshotURL) || !validEvidenceURL(plateCropURL) {
		return domain.DetectionEvent{}, false, NewValidationError(
			"evidence URLs must be empty or start with /evidence/",
		)
	}

	flagged, err := s.watchlist.IsFlagged(ctx, plateKey)
	if err != nil {
		return domain.DetectionEvent{}, false, err
	}
	id, err := s.newID()
	if err != nil {
		return domain.DetectionEvent{}, false, err
	}

	candidate := domain.DetectionEvent{
		ID:             id,
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

	saved, created, err := s.repository.SaveIdempotent(ctx, idempotencyKey, candidate)
	if err != nil {
		return domain.DetectionEvent{}, false, err
	}
	if !created && !sameDetectionRequest(saved, candidate) {
		return domain.DetectionEvent{}, false, NewConflictError(
			"Idempotency-Key was already used for a different detection",
		)
	}

	// Publish both first deliveries and replays. Re-publishing the same event ID lets
	// clients de-duplicate while recovering from a response lost after persistence.
	if err := s.publisher.Publish(ctx, saved); err != nil {
		return domain.DetectionEvent{}, false, err
	}
	return saved, created, nil
}

func (s *DetectionService) List(
	ctx context.Context,
	limit int,
	cursorValue string,
) (DetectionPage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	cursor, err := decodeDetectionCursor(cursorValue)
	if err != nil {
		return DetectionPage{}, NewValidationError("invalid cursor")
	}

	events, err := s.repository.List(ctx, limit+1, cursor)
	if err != nil {
		return DetectionPage{}, err
	}

	page := DetectionPage{Items: events}
	if len(events) <= limit {
		return page, nil
	}

	page.Items = events[:limit]
	last := page.Items[len(page.Items)-1]
	nextCursor, err := encodeDetectionCursor(domain.DetectionCursor{
		DetectedAt: last.DetectedAt,
		ID:         last.ID,
	})
	if err != nil {
		return DetectionPage{}, err
	}
	page.NextCursor = nextCursor
	return page, nil
}

func sameDetectionRequest(existing domain.DetectionEvent, candidate domain.DetectionEvent) bool {
	return existing.CameraID == candidate.CameraID &&
		existing.TrackID == candidate.TrackID &&
		existing.PlateText == candidate.PlateText &&
		existing.PlateKey == candidate.PlateKey &&
		existing.Confidence == candidate.Confidence &&
		existing.SnapshotURL == candidate.SnapshotURL &&
		existing.PlateCropURL == candidate.PlateCropURL
}

func validEvidenceURL(value string) bool {
	return value == "" || strings.HasPrefix(value, "/evidence/")
}

func encodeDetectionCursor(cursor domain.DetectionCursor) (string, error) {
	payload, err := json.Marshal(detectionCursorPayload{
		DetectedAt: cursor.DetectedAt,
		ID:         cursor.ID,
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeDetectionCursor(value string) (*domain.DetectionCursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}

	var decoded detectionCursorPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	if decoded.DetectedAt.IsZero() || strings.TrimSpace(decoded.ID) == "" {
		return nil, NewValidationError("cursor is missing required fields")
	}
	return &domain.DetectionCursor{
		DetectedAt: decoded.DetectedAt,
		ID:         decoded.ID,
	}, nil
}

func randomID() (string, error) {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "evt_" + hex.EncodeToString(bytes[:]), nil
}
