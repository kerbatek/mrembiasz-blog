package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type dynamoUpdater interface {
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

var dynamodbClient dynamoUpdater

type postViewEvent struct {
	PostSlug   string
	ReceivedAt time.Time
}

func parseMessage(record events.SQSMessage, fallbackReceivedAt time.Time) (postViewEvent, error) {
	body := []byte(record.Body)

	var envelope struct {
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Message != "" {
		body = []byte(envelope.Message)
	}

	var message struct {
		PostSlug   string `json:"post_slug"`
		ReceivedAt string `json:"received_at"`
	}
	if err := json.Unmarshal(body, &message); err != nil {
		return postViewEvent{}, err
	}

	postSlug := strings.TrimSpace(message.PostSlug)
	if postSlug == "" {
		return postViewEvent{}, errors.New("missing post_slug")
	}

	receivedAt := fallbackReceivedAt
	if strings.TrimSpace(message.ReceivedAt) != "" {
		var err error
		receivedAt, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(message.ReceivedAt))
		if err != nil {
			return postViewEvent{}, errors.New("invalid received_at")
		}
	}

	return postViewEvent{
		PostSlug:   postSlug,
		ReceivedAt: receivedAt,
	}, nil
}

func updateViewCount(ctx context.Context, client dynamoUpdater, tableName string, postSlug string, now time.Time) error {
	_, err := client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"post_slug": &types.AttributeValueMemberS{Value: postSlug},
		},
		UpdateExpression:    aws.String("SET #updated_at = :updated_at, #updated_at_epoch = :updated_at_epoch ADD #view_count :one"),
		ConditionExpression: aws.String("attribute_not_exists(#updated_at_epoch) OR #updated_at_epoch < :updated_at_epoch"),
		ExpressionAttributeNames: map[string]string{
			"#updated_at":       "updated_at",
			"#updated_at_epoch": "updated_at_epoch",
			"#view_count":       "view_count",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":updated_at":       &types.AttributeValueMemberS{Value: now.UTC().Format(time.RFC3339Nano)},
			":updated_at_epoch": &types.AttributeValueMemberN{Value: strconv.FormatInt(now.UnixNano(), 10)},
			":one":              &types.AttributeValueMemberN{Value: "1"},
		},
	})
	var staleEvent *types.ConditionalCheckFailedException
	if errors.As(err, &staleEvent) {
		_, err = client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String(tableName),
			Key: map[string]types.AttributeValue{
				"post_slug": &types.AttributeValueMemberS{Value: postSlug},
			},
			UpdateExpression: aws.String("ADD #view_count :one"),
			ExpressionAttributeNames: map[string]string{
				"#view_count": "view_count",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":one": &types.AttributeValueMemberN{Value: "1"},
			},
		})
	}
	return err
}

func handleRequest(ctx context.Context, event events.SQSEvent, tableName string, client dynamoUpdater, now time.Time) (events.SQSEventResponse, error) {
	response := events.SQSEventResponse{}

	for _, record := range event.Records {
		postView, err := parseMessage(record, now)
		if err == nil {
			err = updateViewCount(ctx, client, tableName, postView.PostSlug, postView.ReceivedAt)
		}
		if err == nil {
			continue
		}

		log.Printf("failed to process message %s: %v", record.MessageId, err)
		if record.MessageId == "" {
			return response, err
		}
		response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{
			ItemIdentifier: record.MessageId,
		})
	}

	return response, nil
}

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
