package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mrembiasz-blog/backend/internal/appenv"
)

const (
	testAllowedPostSlugs   = `["jenkins-aws-oidc","notes/astro-static","new-post","known-post"]`
	testOriginSecret       = "secret"
	testPostSlugAttribute  = "post_slug"
	testTableName          = "post-views"
	testViewCountAttribute = "view_count"
)

type fakeDynamoDB struct {
	item map[string]types.AttributeValue
	gets []*dynamodb.GetItemInput
}

func (f *fakeDynamoDB) GetItem(_ context.Context, input *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	f.gets = append(f.gets, input)
	return &dynamodb.GetItemOutput{Item: f.item}, nil
}

func viewRequest(slug string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{PathParameters: map[string]string{"slug": slug}}
}

func TestReturnsPostViewCount(t *testing.T) {
	tests := []struct {
		name        string
		slug        string
		wantSlug    string
		storedValue string
		wantBody    string
	}{
		{name: "trimmed slug", slug: " jenkins-aws-oidc ", wantSlug: "jenkins-aws-oidc", storedValue: "42", wantBody: `{"views":42}`},
		{name: "nested slug", slug: "notes/astro-static", wantSlug: "notes/astro-static", storedValue: "7", wantBody: `{"views":7}`},
		{name: "missing counter", slug: "new-post", wantSlug: "new-post", wantBody: `{"views":0}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := map[string]types.AttributeValue{}
			if test.storedValue != "" {
				item[testViewCountAttribute] = &types.AttributeValueMemberN{Value: test.storedValue}
			}
			client := &fakeDynamoDB{item: item}

			result, err := handleRequest(context.Background(), viewRequest(test.slug), testTableName, testAllowedPostSlugs, client)

			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, result.StatusCode)
			assert.JSONEq(t, test.wantBody, result.Body)
			require.Len(t, client.gets, 1)
			assert.Equal(t, testTableName, *client.gets[0].TableName)
			assert.Equal(t, test.wantSlug, client.gets[0].Key[testPostSlugAttribute].(*types.AttributeValueMemberS).Value)
		})
	}
}

func TestRejectsInvalidSlugBeforeDynamoDB(t *testing.T) {
	for _, slug := range []string{"", "unknown-post"} {
		client := &fakeDynamoDB{}

		result, err := handleRequest(context.Background(), viewRequest(slug), testTableName, testAllowedPostSlugs, client)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
		assert.Empty(t, client.gets)
	}
}

func TestRequiresCloudFrontSecret(t *testing.T) {
	client := &fakeDynamoDB{}
	dynamodbClient = client
	t.Cleanup(func() { dynamodbClient = nil })
	t.Setenv(appenv.AnalyticsOriginSecret, testOriginSecret)
	t.Setenv(appenv.PostViewsTableName, testTableName)
	t.Setenv(appenv.ValidPostSlugs, testAllowedPostSlugs)
	request := viewRequest("known-post")

	result, err := lambdaHandler(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, result.StatusCode)
	assert.Empty(t, client.gets)

	request.Headers = map[string]string{"X-Origin-Verify": testOriginSecret}
	result, err = lambdaHandler(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Len(t, client.gets, 1)
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
