package utility

import (
	"bankingApp/internal/api/constants"
	"bankingApp/internal/model"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
	trans    ut.Translator
)

func init() {
	validate = validator.New()
	err := validate.RegisterValidation("isPositive", IsPositive)
	if err != nil {
		slog.Error(err.Error())
	}

	translator := en.New()
	uni := ut.New(translator, translator)
	trans, _ = uni.GetTranslator("en")

	err = validate.RegisterTranslation("isPositive", trans, func(ut ut.Translator) error {
		return ut.Add("isPositive", "{0} must be a positive number", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("isPositive", fe.Field())
		return t
	})
	if err != nil {
		slog.Error(err.Error())
	}

	const required = "required"
	err = validate.RegisterTranslation(required, trans, func(ut ut.Translator) error {
		return ut.Add(required, "{0} is a required field", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(required, fe.Field())
		return t
	})
	if err != nil {
		slog.Error(err.Error())
	}

	const maximum = "max"
	err = validate.RegisterTranslation(maximum, trans, func(ut ut.Translator) error {
		return ut.Add(maximum, "{0} must be at most {1} characters long", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(maximum, fe.Field(), fe.Param())
		return t
	})
	if err != nil {
		slog.Error(err.Error())
	}

	const minimum = "min"
	err = validate.RegisterTranslation(minimum, trans, func(ut ut.Translator) error {
		return ut.Add(minimum, "{0} must be at least {1} characters long", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(minimum, fe.Field(), fe.Param())
		return t
	})
	if err != nil {
		slog.Error(err.Error())
	}

	const oneof = "oneof"
	err = validate.RegisterTranslation(oneof, trans, func(ut ut.Translator) error {
		return ut.Add(oneof, "{0} must be either 'credit' or 'debit'", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(oneof, fe.Field())
		return t
	})
	if err != nil {
		slog.Error(err.Error())
	}
}

func FormulateValidationErrorResponse(errors map[string]string) *APIResponse {
	return &APIResponse{
		Message: constants.BadRequestMessage,
		Errors:  errors,
		Success: false,
	}
}

func HandleValidationErrors(context *gin.Context, validationErrors map[string]string) {
	slog.Info(fmt.Sprintf("validation errors occurred %v", validationErrors))
	context.JSON(http.StatusBadRequest, FormulateValidationErrorResponse(validationErrors))
}

func ValidateRequest(request interface{}) (map[string]string, error) {
	err := validate.Struct(request)
	if err != nil {
		lErr := make(map[string]string)
		for _, dErr := range err.(validator.ValidationErrors) {
			field := dErr.Field()
			fieldStruct, _ := reflect.TypeOf(request).FieldByName(field)
			jsonTag := strings.Split(fieldStruct.Tag.Get("json"), ",")[0]
			lErr[jsonTag] = dErr.Translate(trans)
		}
		return lErr, nil
	}
	return nil, nil
}

func IsPositive(fl validator.FieldLevel) bool {
	val, ok := fl.Field().Interface().(model.Money)
	if !ok {
		return false
	}
	return val.IsPos()
}
