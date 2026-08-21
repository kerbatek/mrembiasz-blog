package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mrembiasz-blog/backend/internal/analytics"
	"mrembiasz-blog/backend/internal/appenv"
)

const (
	testAllowedOrigin    = "https://blog.mrembiasz.pl"
	testAllowedPostSlugs = `["astro-static"]`
	testEventType        = "post_view"
	testOriginSecret     = "secret"
	testPostSlug         = "astro-static"
	testTopicARN         = "topic-arn"
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
	t.Setenv(appenv.ValidPostSlugs, testAllowedPostSlugs)
	client := &fakeSNS{}
	now := time.Date(2026, 8, 17, 12, 30, 0, 123, time.UTC)

	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": " " + testPostSlug + " "},
			Headers: map[string]string{
				"Referer": " " + testAllowedOrigin + "/ ",
			},
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
					SourceIP:  "203.0.113.10",
					UserAgent: "Mozilla/5.0",
				},
			},
		},
		testTopicARN,
		client,
		now,
	)
	require.NoError(t, err)

	assert.Equal(t, http.StatusAccepted, result.StatusCode)
	assert.Equal(t, "application/json", result.Headers["content-type"])
	assert.JSONEq(t, `{"accepted":true}`, result.Body)
	require.Len(t, client.published, 1)
	assert.Equal(t, testTopicARN, *client.published[0].TopicArn)

	var message analytics.Event
	require.NoError(t, json.Unmarshal([]byte(*client.published[0].Message), &message))
	assert.Equal(t, analytics.Event{
		EventType:  testEventType,
		PostSlug:   testPostSlug,
		ReceivedAt: "2026-08-17T12:30:00.000000123Z",
		ClientIP:   "203.0.113.10",
		UserAgent:  "Mozilla/5.0",
		Referer:    testAllowedOrigin + "/",
	}, message)
}

func TestPublishesNestedPostSlug(t *testing.T) {
	t.Setenv(appenv.ValidPostSlugs, `["notes/astro-static"]`)
	client := &fakeSNS{}

	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": "notes/astro-static"},
		},
		testTopicARN,
		client,
		time.Time{},
	)
	require.NoError(t, err)

	assert.Equal(t, http.StatusAccepted, result.StatusCode)
	require.Len(t, client.published, 1)

	var message analytics.Event
	require.NoError(t, json.Unmarshal([]byte(*client.published[0].Message), &message))
	assert.Equal(t, "notes/astro-static", message.PostSlug)
}

func TestRejectsInvalidPayload(t *testing.T) {
	t.Setenv(appenv.ValidPostSlugs, testAllowedPostSlugs)
	tests := []struct {
		name     string
		body     string
		wantBody string
	}{
		{name: "unsupported event", body: `{"event_type":"signup"}`, wantBody: `{"error":"unsupported event_type"}`},
		{name: "invalid JSON", body: `{`, wantBody: `{"error":"invalid JSON body"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := handleRequest(
				context.Background(),
				events.APIGatewayV2HTTPRequest{
					PathParameters: map[string]string{"slug": testPostSlug},
					Body:           test.body,
				},
				testTopicARN,
				&fakeSNS{},
				time.Time{},
			)
			require.NoError(t, err)

			assert.Equal(t, http.StatusBadRequest, result.StatusCode)
			assert.JSONEq(t, test.wantBody, result.Body)
		})
	}
}

func TestRejectsMissingPathParameters(t *testing.T) {
	t.Setenv(appenv.ValidPostSlugs, `[]`)
	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{},
		testTopicARN,
		&fakeSNS{},
		time.Time{},
	)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, result.StatusCode)
	assert.JSONEq(t, `{"error":"missing slug"}`, result.Body)
}

func TestPublishFailureBubblesUp(t *testing.T) {
	t.Setenv(appenv.ValidPostSlugs, testAllowedPostSlugs)
	_, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": testPostSlug},
		},
		testTopicARN,
		&fakeSNS{err: errors.New("sns unavailable")},
		time.Time{},
	)

	require.EqualError(t, err, "sns unavailable")
}

func TestRequiresCloudFrontSecretAndSiteOrigin(t *testing.T) {
	request := events.APIGatewayV2HTTPRequest{Headers: map[string]string{
		"X-Origin-Verify": testOriginSecret,
		"Origin":          testAllowedOrigin,
	}}

	assert.True(t, authorized(request, testOriginSecret, testAllowedOrigin))
	assert.False(t, authorized(request, "wrong", testAllowedOrigin))
	assert.False(t, authorized(request, testOriginSecret, "https://other.example"))
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
