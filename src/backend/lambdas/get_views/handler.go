package main

import (
	"context"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"mrembiasz-blog/backend/internal/httpapi"
)

type dynamoGetter interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

func handleRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest, tableName string, client dynamoGetter) (events.APIGatewayV2HTTPResponse, error) {
	postSlug, err := httpapi.PathParameter(request, "slug")
	if err != nil {
		return httpapi.JSON(400, map[string]any{"error": err.Error()})
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

	response, err := httpapi.JSON(200, map[string]any{"views": viewCount})
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	response.Headers["cache-control"] = "public, max-age=5"
	return response, nil
}
