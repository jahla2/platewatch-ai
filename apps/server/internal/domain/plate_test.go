package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type plateCanonicalizationCase struct {
	Raw string `json:"raw"`
	Key string `json:"key"`
}

func TestSharedCountryAgnosticPlateCanonicalizationCases(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}

	fixturePath := filepath.Join(
		filepath.Dir(currentFile),
		"..",
		"..",
		"..",
		"..",
		"testdata",
		"plate_canonicalization.json",
	)

	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var cases []plateCanonicalizationCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	for _, testCase := range cases {
		t.Run(testCase.Raw, func(t *testing.T) {
			if actual := CanonicalizePlateText(testCase.Raw); actual != testCase.Key {
				t.Fatalf("CanonicalizePlateText(%q) = %q, want %q", testCase.Raw, actual, testCase.Key)
			}
		})
	}
}
