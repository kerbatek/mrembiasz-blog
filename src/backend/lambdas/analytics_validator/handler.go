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
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"mrembiasz-blog/backend/internal/analytics"
	"mrembiasz-blog/backend/internal/httpapi"
)

const maxPostSlugLength = 100

var (
	errInvalidPostConfiguration = errors.New("invalid valid-post configuration")
	postSlugPattern             = regexp.MustCompile(`^[a-z0-9]+([/-][a-z0-9]+)*$`)
)

type analyticsEvent = analytics.Event

type snsPublisher interface {
	Publish(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error)
}

func parsePostSlug(request events.APIGatewayV2HTTPRequest) (string, error) {
	postSlug, err := httpapi.PathParameter(request, "slug")
	if err != nil {
		return "", err
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
	providedSecret := httpapi.Header(request, "x-origin-verify")
	return originSecret != "" && allowedOrigin != "" &&
		subtle.ConstantTimeCompare([]byte(providedSecret), []byte(originSecret)) == 1 &&
		httpapi.Header(request, "origin") == allowedOrigin
}

func parseEventType(request events.APIGatewayV2HTTPRequest) (string, error) {
	eventType := analytics.PostViewEventType
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

	if eventType != analytics.PostViewEventType {
		return "", errors.New("unsupported event_type")
	}

	return eventType, nil
}

func buildAnalyticsEvent(request events.APIGatewayV2HTTPRequest, eventType, postSlug string, now time.Time) analytics.Event {
	return analytics.Event{
		EventType:  eventType,
		PostSlug:   postSlug,
		ReceivedAt: now.UTC().Format(time.RFC3339Nano),
		ClientIP:   strings.TrimSpace(request.RequestContext.HTTP.SourceIP),
		UserAgent:  strings.TrimSpace(request.RequestContext.HTTP.UserAgent),
		Referer:    httpapi.Header(request, "referer"),
	}
}

func handleRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest, topicARN string, client snsPublisher, now time.Time) (events.APIGatewayV2HTTPResponse, error) {
	postSlug, err := parsePostSlug(request)
	if err != nil {
		if errors.Is(err, errInvalidPostConfiguration) {
			return events.APIGatewayV2HTTPResponse{}, err
		}
		return httpapi.JSON(400, map[string]any{"error": err.Error()})
	}

	eventType, err := parseEventType(request)
	if err != nil {
		return httpapi.JSON(400, map[string]any{"error": err.Error()})
	}

	message, err := json.Marshal(buildAnalyticsEvent(request, eventType, postSlug, now))
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	_, err = client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Message:  aws.String(string(message)),
	})
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return httpapi.JSON(202, map[string]any{"accepted": true})
}
