package graphql

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// IsIntrospectionRequest checks if the incoming request looks like a GraphQL introspection query.
// It correctly distinguishes between __type (introspection) and __typename (meta-field).
func IsIntrospectionRequest(r *http.Request) bool {
	if r == nil {
		return false
	}

	// Check URL query parameters first
	query := r.URL.RawQuery
	if strings.Contains(query, "__schema") || containsIntrospectionType(query) || strings.Contains(query, "_service") || strings.Contains(query, "_entities") {
		return true
	}

	// Check request body
	if r.Body == nil {
		return false
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	payload := strings.ToLower(string(body))
	return strings.Contains(payload, "__schema") || strings.Contains(payload, "introspectionquery") || containsIntrospectionType(payload) || strings.Contains(payload, "_service") || strings.Contains(payload, "_entities")
}

// containsIntrospectionType checks for __type but NOT __typename
// In GraphQL introspection, __type is used with parentheses: __type(name: "User")
// While __typename is a field without parentheses
// So we check for __type followed by opening parenthesis
func containsIntrospectionType(s string) bool {
	re := regexp.MustCompile(`__type\s*\(`)
	return re.MatchString(s)
}
