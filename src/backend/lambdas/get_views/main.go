package main

import (
	"context"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"mrembiasz-blog/backend/internal/appenv"
	"mrembiasz-blog/backend/internal/httpapi"
)

var dynamodbClient dynamoGetter

func getDynamoDBClient(ctx context.Context) (dynamoGetter, error) {
	if dynamodbClient != nil {
		return dynamodbClient, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	dynamodbClient = dynamodb.NewFromConfig(cfg)
	return dynamodbClient, nil
}

func lambdaHandler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if !httpapi.HasOriginSecret(request, os.Getenv(appenv.AnalyticsOriginSecret)) {
		return httpapi.JSONError(http.StatusForbidden, "forbidden")
	}

	client, err := getDynamoDBClient(ctx)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return handleRequest(ctx, request, os.Getenv(appenv.PostViewsTableName), os.Getenv(appenv.ValidPostSlugs), client)
}

func main() {
	lambda.Start(lambdaHandler)
}
