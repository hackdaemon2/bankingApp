package nethttp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/monaco-io/request"
)

const PostRequestMethod = "POST"
const GetRequestMethod = "GET"

type RestHttpClient struct {
	Timeout time.Duration
}

// NewRestHttpClient creates a new instance of RestHttpClient
func NewRestHttpClient(timeout time.Duration) *RestHttpClient {
	return &RestHttpClient{
		Timeout: timeout,
	}
}

// GetRequest sends a GET HTTP request to remote resource
func (h *RestHttpClient) GetRequest(
	url string,
	headers map[string]string) (map[string]interface{}, int, error) {
	return h.sendHttpRequest(GetRequestMethod, url, nil, headers)
}

// PostRequest sends a POST HTTP request to remote resource
func (h *RestHttpClient) PostRequest(
	url string,
	request interface{},
	headers map[string]string) (map[string]interface{}, int, error) {
	return h.sendHttpRequest(PostRequestMethod, url, request, headers)
}

// logRequest logs the sent POST HTTP request
func (h *RestHttpClient) logRequest(url string, request interface{}) error {
	slog.Info(fmt.Sprintf("url => %s", url))

	if request == nil {
		slog.Info("method => GET")
		return nil
	}

	slog.Info("method => POST")

	var requestBody []byte
	requestBody, err := json.Marshal(request)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	slog.Info(fmt.Sprintf("Request => %s", string(requestBody)))
	return nil
}

// logResponse logs the response from GET or POST HTTP request
func (h *RestHttpClient) logResponse(response interface{}) error {
	var responseBody []byte
	responseBody, err := json.Marshal(response)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	slog.Info(fmt.Sprintf("Response => %s", string(responseBody)))
	return nil
}

func (h *RestHttpClient) sendHttpRequest(
	method string,
	url string,
	requestBody interface{},
	headers map[string]string) (map[string]interface{}, int, error) {
	var result map[string]interface{}

	client := request.Client{
		URL:     url,
		Method:  method,
		Header:  headers,
		Timeout: h.Timeout,
	}

	var err error

	err = h.logRequest(url, requestBody)
	if err != nil {
		return nil, 0, err
	}

	if requestBody != nil {
		client.JSON = requestBody
	}

	httpRequest := client.Send()
	err = httpRequest.ScanJSON(&result).Error()
	if err != nil {
		return nil, 0, err
	}

	err = h.logResponse(result)
	if err != nil {
		return nil, 0, err
	}

	return result, httpRequest.Response().StatusCode, nil
}
