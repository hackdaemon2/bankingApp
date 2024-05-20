package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ResponseWriterType defines a custom response recorder to capture the status code and response body
type ResponseWriterType struct {
	gin.ResponseWriter
	body *bytes.Buffer
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
			slog.Error("Error reading request body: %v", err.Error())
			context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		context.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		method := strings.ToLower(context.Request.Method)
		uri := context.Request.RequestURI
		if method == "get" || method == "delete" || method == "options" {
			slog.Info(fmt.Sprintf("URI: %s | Method: %s", uri, method))
		} else {
			slog.Info(fmt.Sprintf("URI: %s | Method: %s | Request to Bank Transfer API => %s", uri, string(body), method))
		}
		context.Next()
	}
}

// ResponseLogger logs outgoing responses
func (l *LoggingMiddleware) ResponseLogger() gin.HandlerFunc {
	return func(context *gin.Context) {
		recorder := &ResponseWriterType{ResponseWriter: context.Writer, body: &bytes.Buffer{}}
		context.Writer = recorder
		context.Next()
		responseBody := recorder.body.String()
		var responseMap map[string]interface{}
		if err := json.Unmarshal([]byte(responseBody), &responseMap); err != nil {
			slog.Error("Error decoding response body: %v", err.Error())
		} else {
			slog.Info(fmt.Sprintf("Response from Bank Transfer API => %s", responseBody))
		}
	}
}
