package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"mrembiasz-blog/backend/internal/analytics"
	"mrembiasz-blog/backend/internal/appenv"
	"mrembiasz-blog/backend/internal/httpapi"
)

type snsPublisher interface {
	Publish(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error)
}

func authorized(request events.APIGatewayV2HTTPRequest, originSecret, allowedOrigin string) bool {
	return httpapi.HasOriginSecret(request, originSecret) && allowedOrigin != "" &&
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
	postSlug, err := httpapi.AllowedPostSlug(request, os.Getenv(appenv.ValidPostSlugs))
	if err != nil {
		if errors.Is(err, httpapi.ErrInvalidPostConfiguration) {
			return events.APIGatewayV2HTTPResponse{}, err
		}
		return httpapi.JSONError(http.StatusBadRequest, err.Error())
	}

	eventType, err := parseEventType(request)
	if err != nil {
		return httpapi.JSONError(http.StatusBadRequest, err.Error())
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

	return httpapi.JSON(http.StatusAccepted, map[string]any{"accepted": true})
}
