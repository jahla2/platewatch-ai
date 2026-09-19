package application

import (
	"context"
	"testing"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

type repositoryFake struct {
	saved []domain.DetectionEvent
}

func (r *repositoryFake) Save(_ context.Context, event domain.DetectionEvent) error {
	r.saved = append(r.saved, event)
	return nil
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

func TestCreatePreservesRawTextBuildsKeyAndFlagsPlate(t *testing.T) {
	repository := &repositoryFake{}
	publisher := &publisherFake{}
	service := NewDetectionService(repository, watchlistFake{}, publisher)

	event, err := service.Create(context.Background(), CreateDetectionInput{
		CameraID:     "CAM-01",
		TrackID:      42,
		PlateText:    "abc-1234",
		PlateKey:     "ABC1234",
		Confidence:   0.94,
		SnapshotURL:  "/evidence/CAM-01/42/vehicle.jpg",
		PlateCropURL: "/evidence/CAM-01/42/plate.jpg",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
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

func TestCreateRejectsMismatchedCanonicalKey(t *testing.T) {
	service := NewDetectionService(&repositoryFake{}, watchlistFake{}, &publisherFake{})

	_, err := service.Create(context.Background(), CreateDetectionInput{
		CameraID:   "CAM-01",
		TrackID:    1,
		PlateText:  "ABC-1234",
		PlateKey:   "WRONG",
		Confidence: 0.9,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want plate key mismatch error")
	}
}

func TestCreateRejectsInvalidConfidence(t *testing.T) {
	service := NewDetectionService(&repositoryFake{}, watchlistFake{}, &publisherFake{})
	_, err := service.Create(context.Background(), CreateDetectionInput{
		CameraID: "CAM-01", TrackID: 1, PlateText: "ABC1234", Confidence: 1.5,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
}

func TestCreateRejectsExternalEvidenceURL(t *testing.T) {
	service := NewDetectionService(&repositoryFake{}, watchlistFake{}, &publisherFake{})

	_, err := service.Create(context.Background(), CreateDetectionInput{
		CameraID:    "CAM-01",
		TrackID:     1,
		PlateText:   "ABC1234",
		Confidence:  0.9,
		SnapshotURL: "https://example.com/image.jpg",
	})
	if err == nil {
		t.Fatal("Create() error = nil, want evidence URL validation error")
	}
}
