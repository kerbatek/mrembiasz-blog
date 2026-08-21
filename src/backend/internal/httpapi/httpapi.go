package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

const maxPostSlugLength = 100

var (
	ErrInvalidPostConfiguration = errors.New("invalid valid-post configuration")
	postSlugPattern             = regexp.MustCompile(`^[a-z0-9]+([/-][a-z0-9]+)*$`)
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

func JSONError(statusCode int, message string) (events.APIGatewayV2HTTPResponse, error) {
	return JSON(statusCode, map[string]string{"error": message})
}

func Header(request events.APIGatewayV2HTTPRequest, name string) string {
	for key, value := range request.Headers {
		if strings.EqualFold(key, name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func pathParameter(request events.APIGatewayV2HTTPRequest, name string) (string, error) {
	value := strings.TrimSpace(request.PathParameters[name])
	if value == "" {
		return "", errors.New("missing " + name)
	}
	return value, nil
}

func AllowedPostSlug(request events.APIGatewayV2HTTPRequest, encodedAllowedSlugs string) (string, error) {
	postSlug, err := pathParameter(request, "slug")
	if err != nil {
		return "", err
	}
	if len(postSlug) > maxPostSlugLength || !postSlugPattern.MatchString(postSlug) {
		return "", errors.New("invalid slug")
	}

	var allowedSlugs []string
	if err := json.Unmarshal([]byte(encodedAllowedSlugs), &allowedSlugs); err != nil {
		return "", ErrInvalidPostConfiguration
	}
	for _, allowedSlug := range allowedSlugs {
		if postSlug == allowedSlug {
			return postSlug, nil
		}
	}

	return "", errors.New("unknown slug")
}

func HasOriginSecret(request events.APIGatewayV2HTTPRequest, originSecret string) bool {
	providedSecret := Header(request, "x-origin-verify")
	return originSecret != "" && subtle.ConstantTimeCompare([]byte(providedSecret), []byte(originSecret)) == 1
}
