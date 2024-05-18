package utility

import (
	"bankingApp/internal/api/constants"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/govalues/decimal"
)

type APIResponse struct {
	Message string            `json:"message"`
	Success bool              `json:"success"`
	Errors  map[string]string `json:"errors,omitempty"`
	Data    interface{}       `json:"data,omitempty"`
}

func InternalServerError(context *gin.Context) {
	context.JSON(http.StatusInternalServerError, FormulateErrorResponse("an application error occurred"))
}

func FormulateErrorResponse(message string) *APIResponse {
	return &APIResponse{
		Message: message,
		Success: false,
	}
}

func FormulateSuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Message: constants.SuccessfulTransactionMsg,
		Data:    &data,
		Success: true,
	}
}

func HandleError(context *gin.Context, err error, statusCode int, message string) {
	if err != nil {
		slog.Error(err.Error())
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}
	context.JSON(statusCode, FormulateErrorResponse(message))
}

func Float64ToDecimalHookFunc(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
	if from.Kind() != reflect.Float64 {
		return data, nil
	}

	if to != reflect.TypeOf(decimal.Decimal{}) {
		return data, nil
	}

	floatValue := data.(float64)
	return decimal.NewFromFloat64(floatValue)
}
