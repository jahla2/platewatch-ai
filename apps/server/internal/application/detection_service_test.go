package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

type repositoryFake struct {
	saved       []domain.DetectionEvent
	idempotency map[string]domain.DetectionEvent
}

func newRepositoryFake() *repositoryFake {
	return &repositoryFake{idempotency: map[string]domain.DetectionEvent{}}
}

func (r *repositoryFake) SaveIdempotent(
	_ context.Context,
	key string,
	event domain.DetectionEvent,
) (domain.DetectionEvent, bool, error) {
	if existing, found := r.idempotency[key]; found {
		return existing, false, nil
	}
	event.IdempotencyKey = key
	r.idempotency[key] = event
	r.saved = append(r.saved, event)
	return event, true, nil
}

func (r *repositoryFake) List(
	_ context.Context,
	limit int,
	cursor *domain.DetectionCursor,
) ([]domain.DetectionEvent, error) {
	result := make([]domain.DetectionEvent, 0, limit)
	for _, event := range r.saved {
		if cursor != nil {
			if event.DetectedAt.After(cursor.DetectedAt) {
				continue
			}
			if event.DetectedAt.Equal(cursor.DetectedAt) && event.ID >= cursor.ID {
				continue
			}
		}
		result = append(result, event)
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

type watchlistFake struct{}

func (watchlistFake) IsFlagged(_ context.Context, plateKey string) (bool, error) {
	return plateKey == "ABC1234", nil
}

type publisherFake struct {
	events []domain.DetectionEvent
}

func (p *publisherFake) Publish(_ context.Context, event domain.DetectionEvent) error {
	p.events = append(p.events, event)
	return nil
}

func TestCreatePreservesRawTextBuildsKeyAndFlagsPlate(t *testing.T) {
	repository := newRepositoryFake()
	publisher := &publisherFake{}
	service := NewDetectionService(repository, watchlistFake{}, publisher)

	event, created, err := service.Create(
		context.Background(),
		CreateDetectionInput{
			CameraID:     "CAM-01",
			TrackID:      42,
			PlateText:    "abc-1234",
			PlateKey:     "ABC1234",
			Confidence:   0.94,
			SnapshotURL:  "/evidence/CAM-01/42/vehicle.jpg",
			PlateCropURL: "/evidence/CAM-01/42/plate.jpg",
		},
		"idem-1",
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !created {
		t.Fatal("Create() created = false, want true")
	}
	if event.PlateText != "abc-1234" {
		t.Fatalf("PlateText = %q, want abc-1234", event.PlateText)
	}
	if event.PlateKey != "ABC1234" {
		t.Fatalf("PlateKey = %q, want ABC1234", event.PlateKey)
	}
	if !event.Flagged {
		t.Fatal("Flagged = false, want true")
	}
	if len(repository.saved) != 1 || len(publisher.events) != 1 {
		t.Fatal("event was not saved and published exactly once")
	}
}

func TestCreateReplaysSameEventForDuplicateIdempotencyKey(t *testing.T) {
	repository := newRepositoryFake()
	publisher := &publisherFake{}
	service := NewDetectionService(repository, watchlistFake{}, publisher)

	input := CreateDetectionInput{
		CameraID:   "CAM-01",
		TrackID:    42,
		PlateText:  "ABC-1234",
		Confidence: 0.94,
	}

	first, created, err := service.Create(context.Background(), input, "same-key")
	if err != nil || !created {
		t.Fatalf("first Create() = %#v, %v, %v", first, created, err)
	}

	second, created, err := service.Create(context.Background(), input, "same-key")
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if created {
		t.Fatal("second Create() created = true, want false")
	}
	if first.ID != second.ID {
		t.Fatalf("replayed ID = %q, want %q", second.ID, first.ID)
	}
	if len(repository.saved) != 1 {
		t.Fatalf("saved events = %d, want 1", len(repository.saved))
	}
	if len(publisher.events) != 2 {
		t.Fatalf("published events = %d, want 2 for replay recovery", len(publisher.events))
	}
}

func TestCreateRequiresIdempotencyKey(t *testing.T) {
	service := NewDetectionService(newRepositoryFake(), watchlistFake{}, &publisherFake{})
	_, _, err := service.Create(
		context.Background(),
		CreateDetectionInput{
			CameraID:   "CAM-01",
			TrackID:    1,
			PlateText:  "ABC1234",
			Confidence: 0.9,
		},
		"",
	)
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
}

func TestCreateRejectsMismatchedCanonicalKey(t *testing.T) {
	service := NewDetectionService(newRepositoryFake(), watchlistFake{}, &publisherFake{})

	_, _, err := service.Create(
		context.Background(),
		CreateDetectionInput{
			CameraID:   "CAM-01",
			TrackID:    1,
			PlateText:  "ABC-1234",
			PlateKey:   "WRONG",
			Confidence: 0.9,
		},
		"idem",
	)
	if err == nil {
		t.Fatal("Create() error = nil, want plate key mismatch error")
	}
}

func TestCreateRejectsInvalidConfidence(t *testing.T) {
	service := NewDetectionService(newRepositoryFake(), watchlistFake{}, &publisherFake{})
	_, _, err := service.Create(
		context.Background(),
		CreateDetectionInput{
			CameraID: "CAM-01", TrackID: 1, PlateText: "ABC1234", Confidence: 1.5,
		},
		"idem",
	)
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
}

func TestCreateRejectsExternalEvidenceURL(t *testing.T) {
	service := NewDetectionService(newRepositoryFake(), watchlistFake{}, &publisherFake{})

	_, _, err := service.Create(
		context.Background(),
		CreateDetectionInput{
			CameraID:    "CAM-01",
			TrackID:     1,
			PlateText:   "ABC1234",
			Confidence:  0.9,
			SnapshotURL: "https://example.com/image.jpg",
		},
		"idem",
	)
	if err == nil {
		t.Fatal("Create() error = nil, want evidence URL validation error")
	}
}

func TestListReturnsKeysetCursor(t *testing.T) {
	repository := newRepositoryFake()
	repository.saved = []domain.DetectionEvent{
		{ID: "evt_3", DetectedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)},
		{ID: "evt_2", DetectedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		{ID: "evt_1", DetectedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	service := NewDetectionService(repository, watchlistFake{}, &publisherFake{})

	page, err := service.List(context.Background(), 2, "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.Items) != 2 || page.NextCursor == "" {
		t.Fatalf("page = %#v, want 2 items and next cursor", page)
	}

	next, err := service.List(context.Background(), 2, page.NextCursor)
	if err != nil {
		t.Fatalf("next List() error = %v", err)
	}
	if len(next.Items) != 1 || next.Items[0].ID != "evt_1" {
		t.Fatalf("next page = %#v", next)
	}
}

func TestCreateRejectsIdempotencyKeyReuseWithDifferentPayload(t *testing.T) {
	repository := newRepositoryFake()
	service := NewDetectionService(repository, watchlistFake{}, &publisherFake{})

	_, created, err := service.Create(
		context.Background(),
		CreateDetectionInput{
			CameraID:   "CAM-01",
			TrackID:    42,
			PlateText:  "ABC1234",
			Confidence: 0.9,
		},
		"reused-key",
	)
	if err != nil || !created {
		t.Fatalf("first Create() = created %v, err %v", created, err)
	}

	_, _, err = service.Create(
		context.Background(),
		CreateDetectionInput{
			CameraID:   "CAM-01",
			TrackID:    43,
			PlateText:  "XYZ987",
			Confidence: 0.9,
		},
		"reused-key",
	)
	if err == nil {
		t.Fatal("second Create() error = nil, want conflict")
	}

	var conflict ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("second Create() error = %T, want ConflictError", err)
	}
}
