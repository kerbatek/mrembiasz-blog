package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"mrembiasz-blog/backend/internal/analytics"
	"mrembiasz-blog/backend/internal/httpapi"
)

type dynamoGetter interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

func handleRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest, tableName, allowedPostSlugs string, client dynamoGetter) (events.APIGatewayV2HTTPResponse, error) {
	postSlug, err := httpapi.AllowedPostSlug(request, allowedPostSlugs)
	if err != nil {
		if errors.Is(err, httpapi.ErrInvalidPostConfiguration) {
			return events.APIGatewayV2HTTPResponse{}, err
		}
		return httpapi.JSONError(http.StatusBadRequest, err.Error())
	}

	result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			analytics.PostSlugAttribute: &types.AttributeValueMemberS{Value: postSlug},
		},
		ProjectionExpression: aws.String(analytics.ViewCountAttribute),
	})
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	viewCount := 0
	if value, ok := result.Item[analytics.ViewCountAttribute].(*types.AttributeValueMemberN); ok {
		viewCount, err = strconv.Atoi(value.Value)
		if err != nil {
			return events.APIGatewayV2HTTPResponse{}, err
		}
	}

	return httpapi.JSON(http.StatusOK, map[string]any{"views": viewCount})
}
