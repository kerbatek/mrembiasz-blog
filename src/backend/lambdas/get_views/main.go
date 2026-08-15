package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type dynamoGetter interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

var dynamodbClient dynamoGetter

func response(statusCode int, body map[string]any) (events.APIGatewayV2HTTPResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"cache-control": "public, max-age=5",
			"content-type":  "application/json",
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

func handleRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest, tableName string, client dynamoGetter) (events.APIGatewayV2HTTPResponse, error) {
	postSlug, err := parsePostSlug(request)
	if err != nil {
		return response(400, map[string]any{"error": err.Error()})
	}

	result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"post_slug": &types.AttributeValueMemberS{Value: postSlug},
		},
		ProjectionExpression: aws.String("view_count"),
	})
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	viewCount := 0
	if value, ok := result.Item["view_count"].(*types.AttributeValueMemberN); ok {
		viewCount, err = strconv.Atoi(value.Value)
		if err != nil {
			return events.APIGatewayV2HTTPResponse{}, err
		}
	}

	return response(200, map[string]any{"views": viewCount})
}

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
