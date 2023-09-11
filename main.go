package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/bradfitz/gomemcache/memcache"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
)

type ServerInfo struct {
	API  string `json:"api"`
	WS   string `json:"ws"`
	GRPC string `json:"grpc"`
}

var memcachedClient *memcache.Client

func main() {
	memcachedClient = memcache.New(os.Getenv("MEMCACHED_ADDR"))

	lambda.Start(handleRequest)
}

func handleRequest(_ context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	parts := strings.Split(request.RawPath, "/")
	if len(parts) < 2 {
		return events.LambdaFunctionURLResponse{StatusCode: http.StatusNotFound}, nil
	}

	key := parts[1]

	switch request.RequestContext.HTTP.Method {
	case http.MethodGet:
		return getServerInfo(key)
	case http.MethodPost:
		_, err := memcachedClient.Get(key)
		if err != nil {
			body, err := json.Marshal(
				map[string]string{"message": "Record already exists. Use PATCH method to update"},
			)
			if err != nil {
				return events.LambdaFunctionURLResponse{StatusCode: http.StatusInternalServerError}, errors.New("failed to marshal json")
			}

			return events.LambdaFunctionURLResponse{
				StatusCode: http.StatusForbidden,
				Body:       string(body),
			}, nil
		}

		return updateServerInfo(key, request)
	case http.MethodPatch:
		return updateServerInfo(key, request)
	case http.MethodDelete:
		return deleteServerInfo(key)
	default:
		return events.LambdaFunctionURLResponse{StatusCode: http.StatusMethodNotAllowed}, nil
	}
}

func getServerInfo(key string) (events.LambdaFunctionURLResponse, error) {
	item, err := memcachedClient.Get(key)
	if err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: http.StatusNotFound}, nil
	}

	return events.LambdaFunctionURLResponse{
		Body:       string(item.Value),
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

func updateServerInfo(key string, r events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	serverInfo := ServerInfo{
		API:  r.QueryStringParameters["api"],
		WS:   r.QueryStringParameters["ws"],
		GRPC: r.QueryStringParameters["grpc"],
	}

	jsonResponse, err := json.Marshal(serverInfo)
	if err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: http.StatusInternalServerError}, errors.New("failed to marshal json")
	}

	item := &memcache.Item{
		Key:   key,
		Value: jsonResponse,
	}

	if err = memcachedClient.Set(item); err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: http.StatusInternalServerError}, errors.New("failed to set item to cache")
	}

	return events.LambdaFunctionURLResponse{StatusCode: http.StatusNoContent}, nil
}

func deleteServerInfo(key string) (events.LambdaFunctionURLResponse, error) {
	if err := memcachedClient.Delete(key); err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: http.StatusNotFound}, nil
	}

	return events.LambdaFunctionURLResponse{StatusCode: http.StatusNoContent}, nil
}
