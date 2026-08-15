package main

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDynamoDB struct {
	item map[string]types.AttributeValue
	gets []*dynamodb.GetItemInput
}

func (f *fakeDynamoDB) GetItem(_ context.Context, input *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	f.gets = append(f.gets, input)
	return &dynamodb.GetItemOutput{Item: f.item}, nil
}

func TestReturnsPostViewCount(t *testing.T) {
	client := &fakeDynamoDB{item: map[string]types.AttributeValue{
		"view_count": &types.AttributeValueMemberN{Value: "42"},
	}}

	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": " jenkins-aws-oidc "},
		},
		"post-views",
		client,
	)

	require.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
	assert.JSONEq(t, `{"views":42}`, result.Body)
	require.Len(t, client.gets, 1)
	assert.Equal(t, "post-views", *client.gets[0].TableName)
	assert.Equal(t, "jenkins-aws-oidc", client.gets[0].Key["post_slug"].(*types.AttributeValueMemberS).Value)
}

func TestReturnsNestedPostViewCount(t *testing.T) {
	client := &fakeDynamoDB{item: map[string]types.AttributeValue{
		"view_count": &types.AttributeValueMemberN{Value: "7"},
	}}

	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": "notes/astro-static"},
		},
		"post-views",
		client,
	)

	require.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
	assert.JSONEq(t, `{"views":7}`, result.Body)
	assert.Equal(t, "notes/astro-static", client.gets[0].Key["post_slug"].(*types.AttributeValueMemberS).Value)
}

func TestReturnsZeroForMissingCounter(t *testing.T) {
	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": "new-post"},
		},
		"post-views",
		&fakeDynamoDB{},
	)

	require.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
	assert.JSONEq(t, `{"views":0}`, result.Body)
}

func TestRejectsMissingPostSlug(t *testing.T) {
	client := &fakeDynamoDB{}

	result, err := handleRequest(
		context.Background(),
		events.APIGatewayV2HTTPRequest{
			PathParameters: map[string]string{"slug": ""},
		},
		"post-views",
		client,
	)

	require.NoError(t, err)
	assert.Equal(t, 400, result.StatusCode)
	assert.Empty(t, client.gets)
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
