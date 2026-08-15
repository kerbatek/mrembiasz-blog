package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/sns"
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
	if err != nil {
		t.Fatal(err)
	}

	if result.StatusCode != 202 {
		t.Fatalf("status = %d, want 202", result.StatusCode)
	}
	if result.Headers["content-type"] != "application/json" {
		t.Fatalf("content-type = %q", result.Headers["content-type"])
	}
	if result.Body != `{"accepted":true}` {
		t.Fatalf("body = %s", result.Body)
	}
	if *client.published[0].TopicArn != "topic-arn" {
		t.Fatalf("topic arn = %q", *client.published[0].TopicArn)
	}

	var message map[string]string
	if err := json.Unmarshal([]byte(*client.published[0].Message), &message); err != nil {
		t.Fatal(err)
	}
	if message["post_slug"] != "astro-static" {
		t.Fatalf("post_slug = %q", message["post_slug"])
	}
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
	if err != nil {
		t.Fatal(err)
	}

	if result.StatusCode != 202 {
		t.Fatalf("status = %d, want 202", result.StatusCode)
	}

	var message map[string]string
	if err := json.Unmarshal([]byte(*client.published[0].Message), &message); err != nil {
		t.Fatal(err)
	}
	if message["post_slug"] != "notes/astro-static" {
		t.Fatalf("post_slug = %q", message["post_slug"])
	}
}

func TestRejectsMissingPathParameters(t *testing.T) {
	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{},
		"topic-arn",
		&fakeSNS{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.StatusCode != 400 {
		t.Fatalf("status = %d, want 400", result.StatusCode)
	}
	if result.Body != `{"error":"missing slug"}` {
		t.Fatalf("body = %s", result.Body)
	}
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
	if err != nil {
		t.Fatal(err)
	}

	if result.StatusCode != 400 {
		t.Fatalf("status = %d, want 400", result.StatusCode)
	}
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

	if err == nil || err.Error() != "sns unavailable" {
		t.Fatalf("err = %v, want sns unavailable", err)
	}
}

func TestReusesSNSClientAcrossWarmInvocations(t *testing.T) {
	existing := &fakeSNS{}
	publisher = existing
	t.Cleanup(func() { publisher = nil })

	first, err := getPublisher(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := getPublisher(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if first != existing || second != existing {
		t.Fatal("publisher was not reused")
	}
}
