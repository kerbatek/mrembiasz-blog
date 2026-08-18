package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
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
	client, err := getDynamoDBClient(ctx)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return handleRequest(ctx, request, os.Getenv("POST_VIEWS_TABLE_NAME"), client)
}

func main() {
	lambda.Start(lambdaHandler)
}
