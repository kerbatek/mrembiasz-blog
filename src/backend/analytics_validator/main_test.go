package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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
	client := &fakeSNS{}

	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": " astro-static "},
		},
		"topic-arn",
		client,
	)
	require.NoError(t, err)

	assert.Equal(t, 202, result.StatusCode)
	assert.Equal(t, "application/json", result.Headers["content-type"])
	assert.JSONEq(t, `{"accepted":true}`, result.Body)
	require.Len(t, client.published, 1)
	assert.Equal(t, "topic-arn", *client.published[0].TopicArn)

	var message map[string]string
	require.NoError(t, json.Unmarshal([]byte(*client.published[0].Message), &message))
	assert.Equal(t, "astro-static", message["post_slug"])
}

func TestPublishesNestedPostSlug(t *testing.T) {
	client := &fakeSNS{}

	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": "notes/astro-static"},
		},
		"topic-arn",
		client,
	)
	require.NoError(t, err)

	assert.Equal(t, 202, result.StatusCode)
	require.Len(t, client.published, 1)

	var message map[string]string
	require.NoError(t, json.Unmarshal([]byte(*client.published[0].Message), &message))
	assert.Equal(t, "notes/astro-static", message["post_slug"])
}

func TestRejectsMissingPathParameters(t *testing.T) {
	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{},
		"topic-arn",
		&fakeSNS{},
	)
	require.NoError(t, err)

	assert.Equal(t, 400, result.StatusCode)
	assert.JSONEq(t, `{"error":"missing slug"}`, result.Body)
}

func TestRejectsMissingPostSlug(t *testing.T) {
	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": ""},
		},
		"topic-arn",
		&fakeSNS{},
	)
	require.NoError(t, err)

	assert.Equal(t, 400, result.StatusCode)
}

func TestPublishFailureBubblesToAPIGatewayRetryOr500(t *testing.T) {
	_, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": "astro-static"},
		},
		"topic-arn",
		&fakeSNS{err: errors.New("sns unavailable")},
	)

	require.EqualError(t, err, "sns unavailable")
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
