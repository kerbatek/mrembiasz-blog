package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type snsPublisher interface {
	Publish(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error)
}

var publisher snsPublisher

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

	return postSlug, nil
}

func handleRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest, topicARN string, client snsPublisher) (events.APIGatewayV2HTTPResponse, error) {
	postSlug, err := parsePostSlug(request)
	if err != nil {
		return response(400, map[string]any{"error": err.Error()})
	}

	message, err := json.Marshal(map[string]string{"post_slug": postSlug})
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
	client, err := getPublisher(ctx)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return handleRequest(ctx, request, os.Getenv("ANALYTICS_TOPIC_ARN"), client)
}

func stringPtr(value string) *string {
	return &value
}

func main() {
	lambda.Start(lambdaHandler)
}
