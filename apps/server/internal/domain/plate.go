package domain

import (
	"regexp"
	"strings"
)

var nonAlphaNumeric = regexp.MustCompile(`[^A-Z0-9]`)

func NormalizePlate(value string) string {
	return nonAlphaNumeric.ReplaceAllString(strings.ToUpper(strings.TrimSpace(value)), "")
}
