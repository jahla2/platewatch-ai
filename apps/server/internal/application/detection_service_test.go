package application

import (
	"context"
	"testing"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

type repositoryFake struct {
	saved []domain.DetectionEvent
	keys  map[string]struct{}
}

func newRepositoryFake() *repositoryFake {
	return &repositoryFake{keys: map[string]struct{}{}}
}

func (r *repositoryFake) SaveIfAbsent(
	_ context.Context,
	event domain.DetectionEvent,
) (bool, error) {
	if _, exists := r.keys[event.IdempotencyKey]; exists {
		return false, nil
	}
	r.keys[event.IdempotencyKey] = struct{}{}
	r.saved = append(r.saved, event)
	return true, nil
}

func (r *repositoryFake) List(_ context.Context, _ int) ([]domain.DetectionEvent, error) {
	return r.saved, nil
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

func validDetectionInput() CreateDetectionInput {
	return CreateDetectionInput{
		IdempotencyKey: "CAM-01:42:ABC1234",
		CameraID:       "CAM-01",
		TrackID:        42,
		PlateText:      "abc-1234",
		PlateKey:       "ABC1234",
		Confidence:     0.94,
		SnapshotURL:    "/evidence/CAM-01/42/vehicle.jpg",
		PlateCropURL:   "/evidence/CAM-01/42/plate.jpg",
	}
}

func TestCreatePreservesRawTextBuildsKeyAndFlagsPlate(t *testing.T) {
	repository := newRepositoryFake()
	publisher := &publisherFake{}
	service := NewDetectionService(repository, watchlistFake{}, publisher)

	event, created, err := service.Create(context.Background(), validDetectionInput())
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

func TestCreateIsIdempotentAndDoesNotRepublish(t *testing.T) {
	repository := newRepositoryFake()
	publisher := &publisherFake{}
	service := NewDetectionService(repository, watchlistFake{}, publisher)
	input := validDetectionInput()

	first, firstCreated, err := service.Create(context.Background(), input)
	if err != nil || !firstCreated {
		t.Fatalf("first Create() = %#v, %v, %v", first, firstCreated, err)
	}

	second, secondCreated, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if secondCreated {
		t.Fatal("second Create() created = true, want false")
	}
	if first.ID != second.ID {
		t.Fatalf("duplicate event ID changed: %q != %q", first.ID, second.ID)
	}
	if len(repository.saved) != 1 || len(publisher.events) != 1 {
		t.Fatal("duplicate request persisted or published more than once")
	}
}

func TestCreateRejectsMissingIdempotencyKey(t *testing.T) {
	service := NewDetectionService(newRepositoryFake(), watchlistFake{}, &publisherFake{})
	input := validDetectionInput()
	input.IdempotencyKey = ""

	_, _, err := service.Create(context.Background(), input)
	if err == nil {
		t.Fatal("Create() error = nil, want idempotency validation error")
	}
}

func TestCreateRejectsMismatchedCanonicalKey(t *testing.T) {
	service := NewDetectionService(newRepositoryFake(), watchlistFake{}, &publisherFake{})
	input := validDetectionInput()
	input.PlateKey = "WRONG"

	_, _, err := service.Create(context.Background(), input)
	if err == nil {
		t.Fatal("Create() error = nil, want plate key mismatch error")
	}
}

func TestCreateRejectsInvalidConfidence(t *testing.T) {
	service := NewDetectionService(newRepositoryFake(), watchlistFake{}, &publisherFake{})
	input := validDetectionInput()
	input.Confidence = 1.5

	_, _, err := service.Create(context.Background(), input)
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
}

func TestCreateRejectsExternalEvidenceURL(t *testing.T) {
	service := NewDetectionService(newRepositoryFake(), watchlistFake{}, &publisherFake{})
	input := validDetectionInput()
	input.SnapshotURL = "https://example.com/image.jpg"

	_, _, err := service.Create(context.Background(), input)
	if err == nil {
		t.Fatal("Create() error = nil, want evidence URL validation error")
	}
}
