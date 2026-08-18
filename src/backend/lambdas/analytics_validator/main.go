package main

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"mrembiasz-blog/backend/internal/httpapi"
)

var publisher snsPublisher

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
		return httpapi.JSON(403, map[string]any{"error": "forbidden"})
	}

	client, err := getPublisher(ctx)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return handleRequest(ctx, request, os.Getenv("ANALYTICS_TOPIC_ARN"), client, time.Now())
}

func main() {
	lambda.Start(lambdaHandler)
}
