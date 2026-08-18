package main

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var dynamodbClient dynamoUpdater

func getDynamoDBClient(ctx context.Context) (dynamoUpdater, error) {
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

func lambdaHandler(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
	client, err := getDynamoDBClient(ctx)
	if err != nil {
		return events.SQSEventResponse{}, err
	}

	return handleRequest(ctx, event, os.Getenv("POST_VIEWS_TABLE_NAME"), client, time.Now())
}

func main() {
	lambda.Start(lambdaHandler)
}
