package middleware

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
)

// ResponseWriterType defines a custom response recorder to capture the status code and response body
type ResponseWriterType struct {
	gin.ResponseWriter
	status int
	body   *bytes.Buffer // Buffer to hold response body
}

func (r *ResponseWriterType) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ResponseWriterType) Write(data []byte) (int, error) {
	if r.body == nil {
		r.body = &bytes.Buffer{}
	}
	r.body.Write(data)
	return r.ResponseWriter.Write(data)
}

// LoggingMiddleware handles logging requests and responses
type LoggingMiddleware struct{}

// RequestLogger logs incoming requests
func (l *LoggingMiddleware) RequestLogger() gin.HandlerFunc {
	return func(context *gin.Context) {
		body, err := io.ReadAll(context.Request.Body)
		if err != nil {
			slog.Error(err.Error())
		}
		context.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		jsonBody := string(body)
		data := []byte(strings.ReplaceAll(strings.ReplaceAll(jsonBody, "\n", ""), " ", ""))
		slog.Info(fmt.Sprintf("Request to Bank Transfer API => %s", data))
		context.Next()
	}
}

// ResponseLogger logs outgoing responses
func (l *LoggingMiddleware) ResponseLogger() gin.HandlerFunc {
	return func(context *gin.Context) {
		recorder := &ResponseWriterType{ResponseWriter: context.Writer}
		context.Writer = recorder
		context.Next()
		responseBody := recorder.body.String()
		var responseMap map[string]interface{}
		err := mapstructure.Decode(responseBody, responseMap)
		if err != nil {
			slog.Error(err.Error())
		}
		slog.Info(fmt.Sprintf("Response from Bank Transfer API => %s", responseBody))
	}
}
