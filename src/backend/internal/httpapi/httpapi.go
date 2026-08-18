package httpapi

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

func JSON(statusCode int, body any) (events.APIGatewayV2HTTPResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       string(payload),
	}, nil
}

func Header(request events.APIGatewayV2HTTPRequest, name string) string {
	for key, value := range request.Headers {
		if strings.EqualFold(key, name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func PathParameter(request events.APIGatewayV2HTTPRequest, name string) (string, error) {
	value := strings.TrimSpace(request.PathParameters[name])
	if value == "" {
		return "", errors.New("missing " + name)
	}
	return value, nil
}
