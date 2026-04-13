package graphql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_matchesSkipPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		urlPath string
		want    bool
	}{
		{"exact_match", "/health", "/health", true},
		{"exact_no_match", "/health", "/healthz", false},
		{"wildcard_match", "/webhooks/*", "/webhooks/stripe", true},
		{"wildcard_match_nested", "/webhooks/*", "/webhooks/stripe/events", true},
		{"wildcard_no_match", "/webhooks/*", "/api/webhooks/stripe", false},
		{"wildcard_base_only", "/webhooks/*", "/webhooks", false},
		{"bare_wildcard", "/*", "/anything", true},
		{"empty_pattern", "", "/path", false},
		{"empty_path", "/health", "", false},
		{"both_empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			got := matchesSkipPath(tt.pattern, tt.urlPath)

			// Assert
			assert.Equal(t, tt.want, got)
		})
	}
}
