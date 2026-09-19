package domain

import (
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

var plateUpper = cases.Upper(language.Und)

func CanonicalizePlateText(value string) string {
	normalized := plateUpper.String(norm.NFKC.String(strings.TrimSpace(value)))

	var builder strings.Builder
	builder.Grow(len(normalized))

	for _, character := range normalized {
		if unicode.IsLetter(character) || unicode.IsNumber(character) {
			builder.WriteRune(character)
		}
	}

	return builder.String()
}
