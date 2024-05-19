package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

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
			slog.Error("Error reading request body: %v", err)
			context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		context.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		slog.Info(fmt.Sprintf("Request to Bank Transfer API => %s", string(body)))

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
			slog.Error("Error decoding response body: %v", err)
		} else {
			slog.Info(fmt.Sprintf("Response from Bank Transfer API => %s", responseBody))
		}
	}
}
