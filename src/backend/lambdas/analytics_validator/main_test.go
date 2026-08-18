package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSNS struct {
	published []*sns.PublishInput
	err       error
}

func (f *fakeSNS) Publish(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
	f.published = append(f.published, input)
	return &sns.PublishOutput{}, f.err
}

func TestPublishesValidPostViewEvent(t *testing.T) {
	t.Setenv("VALID_POST_SLUGS", `["astro-static"]`)
	client := &fakeSNS{}
	now := time.Date(2026, 8, 17, 12, 30, 0, 123, time.UTC)

	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": " astro-static "},
			Headers: map[string]string{
				"Referer": " https://blog.mrembiasz.pl/ ",
			},
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
					SourceIP:  "203.0.113.10",
					UserAgent: "Mozilla/5.0",
				},
			},
		},
		"topic-arn",
		client,
		now,
	)
	require.NoError(t, err)

	assert.Equal(t, 202, result.StatusCode)
	assert.Equal(t, "application/json", result.Headers["content-type"])
	assert.JSONEq(t, `{"accepted":true}`, result.Body)
	require.Len(t, client.published, 1)
	assert.Equal(t, "topic-arn", *client.published[0].TopicArn)

	var message analyticsEvent
	require.NoError(t, json.Unmarshal([]byte(*client.published[0].Message), &message))
	assert.Equal(t, analyticsEvent{
		EventType:  "post_view",
		PostSlug:   "astro-static",
		ReceivedAt: "2026-08-17T12:30:00.000000123Z",
		ClientIP:   "203.0.113.10",
		UserAgent:  "Mozilla/5.0",
		Referer:    "https://blog.mrembiasz.pl/",
	}, message)
}

func TestPublishesNestedPostSlug(t *testing.T) {
	t.Setenv("VALID_POST_SLUGS", `["notes/astro-static"]`)
	client := &fakeSNS{}

	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": "notes/astro-static"},
		},
		"topic-arn",
		client,
		time.Time{},
	)
	require.NoError(t, err)

	assert.Equal(t, 202, result.StatusCode)
	require.Len(t, client.published, 1)

	var message analyticsEvent
	require.NoError(t, json.Unmarshal([]byte(*client.published[0].Message), &message))
	assert.Equal(t, "notes/astro-static", message.PostSlug)
}

func TestRejectsUnsupportedEventType(t *testing.T) {
	t.Setenv("VALID_POST_SLUGS", `["astro-static"]`)
	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": "astro-static"},
			Body:           `{"event_type":"signup"}`,
		},
		"topic-arn",
		&fakeSNS{},
		time.Time{},
	)
	require.NoError(t, err)

	assert.Equal(t, 400, result.StatusCode)
	assert.JSONEq(t, `{"error":"unsupported event_type"}`, result.Body)
}

func TestRejectsInvalidJSONBody(t *testing.T) {
	t.Setenv("VALID_POST_SLUGS", `["astro-static"]`)
	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": "astro-static"},
			Body:           `{`,
		},
		"topic-arn",
		&fakeSNS{},
		time.Time{},
	)
	require.NoError(t, err)

	assert.Equal(t, 400, result.StatusCode)
	assert.JSONEq(t, `{"error":"invalid JSON body"}`, result.Body)
}

func TestRejectsMissingPathParameters(t *testing.T) {
	t.Setenv("VALID_POST_SLUGS", `[]`)
	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{},
		"topic-arn",
		&fakeSNS{},
		time.Time{},
	)
	require.NoError(t, err)

	assert.Equal(t, 400, result.StatusCode)
	assert.JSONEq(t, `{"error":"missing slug"}`, result.Body)
}

func TestRejectsMissingPostSlug(t *testing.T) {
	t.Setenv("VALID_POST_SLUGS", `[]`)
	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": ""},
		},
		"topic-arn",
		&fakeSNS{},
		time.Time{},
	)
	require.NoError(t, err)

	assert.Equal(t, 400, result.StatusCode)
}

func TestPublishFailureBubblesToAPIGatewayRetryOr500(t *testing.T) {
	t.Setenv("VALID_POST_SLUGS", `["astro-static"]`)
	_, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": "astro-static"},
		},
		"topic-arn",
		&fakeSNS{err: errors.New("sns unavailable")},
		time.Time{},
	)

	require.EqualError(t, err, "sns unavailable")
}

func TestRejectsInvalidAndUnknownSlugs(t *testing.T) {
	t.Setenv("VALID_POST_SLUGS", `["known-post"]`)

	for _, slug := range []string{"UPPERCASE", "../known-post", strings.Repeat("a", maxPostSlugLength+1), "unknown-post"} {
		result, err := handleRequest(context.Background(), events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": slug},
		}, "topic-arn", &fakeSNS{}, time.Time{})
		require.NoError(t, err)
		assert.Equal(t, 400, result.StatusCode)
	}
}

func TestRequiresCloudFrontSecretAndSiteOrigin(t *testing.T) {
	request := events.APIGatewayV2HTTPRequest{Headers: map[string]string{
		"X-Origin-Verify": "secret",
		"Origin":          "https://blog.mrembiasz.pl",
	}}

	assert.True(t, authorized(request, "secret", "https://blog.mrembiasz.pl"))
	assert.False(t, authorized(request, "wrong", "https://blog.mrembiasz.pl"))
	assert.False(t, authorized(request, "secret", "https://other.example"))
}

func TestReusesSNSClientAcrossWarmInvocations(t *testing.T) {
	existing := &fakeSNS{}
	publisher = existing
	t.Cleanup(func() { publisher = nil })

	first, err := getPublisher(context.Background())
	require.NoError(t, err)
	second, err := getPublisher(context.Background())
	require.NoError(t, err)

	assert.Same(t, existing, first)
	assert.Same(t, existing, second)
}
