package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type snsPublisher interface {
	Publish(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error)
}

var publisher snsPublisher
var errInvalidPostConfiguration = errors.New("invalid valid-post configuration")

const postViewEventType = "post_view"

const maxPostSlugLength = 100

var postSlugPattern = regexp.MustCompile(`^[a-z0-9]+([/-][a-z0-9]+)*$`)

type analyticsEvent struct {
	EventType  string `json:"event_type"`
	PostSlug   string `json:"post_slug"`
	ReceivedAt string `json:"received_at"`
	ClientIP   string `json:"client_ip,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	Referer    string `json:"referer,omitempty"`
}

func response(statusCode int, body map[string]any) (events.APIGatewayV2HTTPResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"content-type": "application/json",
		},
		Body: string(payload),
	}, nil
}

func parsePostSlug(request events.APIGatewayV2HTTPRequest) (string, error) {
	postSlug := strings.TrimSpace(request.PathParameters["slug"])
	if postSlug == "" {
		return "", errors.New("missing slug")
	}
	if len(postSlug) > maxPostSlugLength || !postSlugPattern.MatchString(postSlug) {
		return "", errors.New("invalid slug")
	}

	var validPostSlugs []string
	if err := json.Unmarshal([]byte(os.Getenv("VALID_POST_SLUGS")), &validPostSlugs); err != nil {
		return "", errInvalidPostConfiguration
	}
	for _, validPostSlug := range validPostSlugs {
		if postSlug == validPostSlug {
			return postSlug, nil
		}
	}

	return "", errors.New("unknown slug")
}

func authorized(request events.APIGatewayV2HTTPRequest, originSecret, allowedOrigin string) bool {
	providedSecret := header(request, "x-origin-verify")
	return originSecret != "" && allowedOrigin != "" &&
		subtle.ConstantTimeCompare([]byte(providedSecret), []byte(originSecret)) == 1 &&
		header(request, "origin") == allowedOrigin
}

func parseEventType(request events.APIGatewayV2HTTPRequest) (string, error) {
	eventType := postViewEventType
	if strings.TrimSpace(request.Body) != "" {
		var body struct {
			EventType string `json:"event_type"`
		}
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return "", errors.New("invalid JSON body")
		}
		if strings.TrimSpace(body.EventType) != "" {
			eventType = strings.TrimSpace(body.EventType)
		}
	}

	if eventType != postViewEventType {
		return "", errors.New("unsupported event_type")
	}

	return eventType, nil
}

func header(request events.APIGatewayV2HTTPRequest, name string) string {
	for key, value := range request.Headers {
		if strings.EqualFold(key, name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func buildAnalyticsEvent(request events.APIGatewayV2HTTPRequest, eventType string, postSlug string, now time.Time) analyticsEvent {
	return analyticsEvent{
		EventType:  eventType,
		PostSlug:   postSlug,
		ReceivedAt: now.UTC().Format(time.RFC3339Nano),
		ClientIP:   strings.TrimSpace(request.RequestContext.HTTP.SourceIP),
		UserAgent:  strings.TrimSpace(request.RequestContext.HTTP.UserAgent),
		Referer:    header(request, "referer"),
	}
}

func handleRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest, topicARN string, client snsPublisher, now time.Time) (events.APIGatewayV2HTTPResponse, error) {
	postSlug, err := parsePostSlug(request)
	if err != nil {
		if errors.Is(err, errInvalidPostConfiguration) {
			return events.APIGatewayV2HTTPResponse{}, err
		}
		return response(400, map[string]any{"error": err.Error()})
	}

	eventType, err := parseEventType(request)
	if err != nil {
		return response(400, map[string]any{"error": err.Error()})
	}

	message, err := json.Marshal(buildAnalyticsEvent(request, eventType, postSlug, now))
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	_, err = client.Publish(ctx, &sns.PublishInput{
		TopicArn: &topicARN,
		Message:  stringPtr(string(message)),
	})
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return response(202, map[string]any{"accepted": true})
}

func getPublisher(ctx context.Context) (snsPublisher, error) {
	if publisher != nil {
		return publisher, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	publisher = sns.NewFromConfig(cfg)
	return publisher, nil
}

func lambdaHandler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if !authorized(request, os.Getenv("ANALYTICS_ORIGIN_SECRET"), os.Getenv("ANALYTICS_ALLOWED_ORIGIN")) {
		return response(403, map[string]any{"error": "forbidden"})
	}

	client, err := getPublisher(ctx)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return handleRequest(ctx, request, os.Getenv("ANALYTICS_TOPIC_ARN"), client, time.Now())
}

func stringPtr(value string) *string {
	return &value
}

func main() {
	lambda.Start(lambdaHandler)
}
