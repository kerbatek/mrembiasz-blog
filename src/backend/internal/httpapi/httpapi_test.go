package httpapi

import (
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAllowedPostSlugs = `["known-post","notes/nested-post"]`
	testOriginSecret     = "secret"
)

func requestWithSlug(slug string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{PathParameters: map[string]string{"slug": slug}}
}

func TestAllowedPostSlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		want    string
		wantErr string
	}{
		{name: "known", slug: " known-post ", want: "known-post"},
		{name: "nested", slug: "notes/nested-post", want: "notes/nested-post"},
		{name: "missing", wantErr: "missing slug"},
		{name: "traversal", slug: "../known-post", wantErr: "invalid slug"},
		{name: "uppercase", slug: "KNOWN-POST", wantErr: "invalid slug"},
		{name: "too long", slug: strings.Repeat("a", maxPostSlugLength+1), wantErr: "invalid slug"},
		{name: "unknown", slug: "unknown-post", wantErr: "unknown slug"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := AllowedPostSlug(requestWithSlug(test.slug), testAllowedPostSlugs)
			if test.wantErr != "" {
				require.EqualError(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestAllowedPostSlugRejectsInvalidConfiguration(t *testing.T) {
	_, err := AllowedPostSlug(requestWithSlug("known-post"), "invalid")
	assert.True(t, errors.Is(err, ErrInvalidPostConfiguration))
}

func TestHasOriginSecret(t *testing.T) {
	request := events.APIGatewayV2HTTPRequest{Headers: map[string]string{"X-Origin-Verify": testOriginSecret}}

	assert.True(t, HasOriginSecret(request, testOriginSecret))
	assert.False(t, HasOriginSecret(request, "badbad"))
	assert.False(t, HasOriginSecret(request, ""))
}
