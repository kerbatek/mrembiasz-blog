package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	expectedIncrementExpression = "ADD #view_count :one"
	expectedPostSlugAttribute   = "post_slug"
	testTableName               = "post-views"
)

type fakeDynamoDB struct {
	updates []*dynamodb.UpdateItemInput
	err     error
	errs    []error
}

func (f *fakeDynamoDB) UpdateItem(_ context.Context, input *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	f.updates = append(f.updates, input)
	if len(f.errs) >= len(f.updates) {
		return &dynamodb.UpdateItemOutput{}, f.errs[len(f.updates)-1]
	}
	return &dynamodb.UpdateItemOutput{}, f.err
}

func updatedAt(input *dynamodb.UpdateItemInput) string {
	return input.ExpressionAttributeValues[":updated_at"].(*types.AttributeValueMemberS).Value
}

func sqsEvent(messageID, body string) events.SQSEvent {
	return events.SQSEvent{Records: []events.SQSMessage{{MessageId: messageID, Body: body}}}
}

func TestUpdatesPostViewCounterFromRawSQSMessage(t *testing.T) {
	client := &fakeDynamoDB{}
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

	result, err := handleRequest(
		context.Background(),
		sqsEvent("message-1", `{"post_slug":" jenkins-aws-oidc "}`),
		testTableName,
		client,
		now,
	)

	require.NoError(t, err)
	assert.Empty(t, result.BatchItemFailures)
	require.Len(t, client.updates, 1)
	assert.Equal(t, testTableName, *client.updates[0].TableName)
	assert.Equal(t, "jenkins-aws-oidc", client.updates[0].Key[expectedPostSlugAttribute].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "2026-08-14T00:00:00Z", updatedAt(client.updates[0]))
	assert.Contains(t, *client.updates[0].UpdateExpression, expectedIncrementExpression)
}

func TestUnwrapsSNSMessageFromSQSBody(t *testing.T) {
	client := &fakeDynamoDB{}

	result, err := handleRequest(
		context.Background(),
		sqsEvent("message-1", `{"Message":"{\"post_slug\":\"astro-static\",\"received_at\":\"2026-08-17T12:30:00.000000123Z\"}"}`),
		testTableName,
		client,
		time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC),
	)

	require.NoError(t, err)
	assert.Empty(t, result.BatchItemFailures)
	require.Len(t, client.updates, 1)
	assert.Equal(t, "astro-static", client.updates[0].Key[expectedPostSlugAttribute].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "2026-08-17T12:30:00.000000123Z", updatedAt(client.updates[0]))
}

func TestOlderEventIncrementsWithoutReplacingUpdatedAt(t *testing.T) {
	client := &fakeDynamoDB{errs: []error{&types.ConditionalCheckFailedException{}, nil}}

	result, err := handleRequest(
		context.Background(),
		sqsEvent("older-event", `{"post_slug":"astro-static","received_at":"2026-08-17T12:30:00Z"}`),
		testTableName,
		client,
		time.Now(),
	)

	require.NoError(t, err)
	assert.Empty(t, result.BatchItemFailures)
	require.Len(t, client.updates, 2)
	assert.Contains(t, *client.updates[0].ConditionExpression, "#updated_at_epoch < :updated_at_epoch")
	assert.Equal(t, expectedIncrementExpression, *client.updates[1].UpdateExpression)
}

func TestDropsMalformedSNSMessage(t *testing.T) {
	client := &fakeDynamoDB{}

	result, err := handleRequest(
		context.Background(),
		sqsEvent("bad-sns", `{"Message":"{"}`),
		testTableName,
		client,
		time.Now(),
	)

	require.NoError(t, err)
	assert.Empty(t, result.BatchItemFailures)
	assert.Empty(t, client.updates)
}

func TestDropsInvalidReceivedAt(t *testing.T) {
	client := &fakeDynamoDB{}

	result, err := handleRequest(
		context.Background(),
		sqsEvent("bad-time", `{"post_slug":"valid-post","received_at":"not-a-time"}`),
		testTableName,
		client,
		time.Now(),
	)

	require.NoError(t, err)
	assert.Empty(t, result.BatchItemFailures)
	assert.Empty(t, client.updates)
}

func TestDropsInvalidAndProcessesValidRecords(t *testing.T) {
	client := &fakeDynamoDB{}

	result, err := handleRequest(
		context.Background(),
		events.SQSEvent{Records: []events.SQSMessage{
			{
				MessageId: "good",
				Body:      `{"post_slug":"valid-post"}`,
			},
			{
				MessageId: "bad",
				Body:      `{"post_slug":""}`,
			},
		}},
		testTableName,
		client,
		time.Now(),
	)

	require.NoError(t, err)
	assert.Empty(t, result.BatchItemFailures)
	assert.Len(t, client.updates, 1)
}

func TestReportsDynamoDBFailureAsFailedRecord(t *testing.T) {
	result, err := handleRequest(
		context.Background(),
		sqsEvent("retry-me", `{"post_slug":"valid-post"}`),
		testTableName,
		&fakeDynamoDB{err: errors.New("throttled")},
		time.Now(),
	)

	require.NoError(t, err)
	assert.Equal(t, []events.SQSBatchItemFailure{{ItemIdentifier: "retry-me"}}, result.BatchItemFailures)
}

func TestDropsInvalidRecordWithoutMessageID(t *testing.T) {
	result, err := handleRequest(
		context.Background(),
		sqsEvent("", `{"post_slug":""}`),
		testTableName,
		&fakeDynamoDB{},
		time.Now(),
	)

	require.NoError(t, err)
	assert.Empty(t, result.BatchItemFailures)
}

func TestReusesDynamoDBClientAcrossWarmInvocations(t *testing.T) {
	existing := &fakeDynamoDB{}
	dynamodbClient = existing
	t.Cleanup(func() { dynamodbClient = nil })

	first, err := getDynamoDBClient(context.Background())
	require.NoError(t, err)
	second, err := getDynamoDBClient(context.Background())
	require.NoError(t, err)

	assert.Same(t, existing, first)
	assert.Same(t, existing, second)
}
