package handler_test

import (
	handler "bankingApp/internal/api/handlers"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// tests

func Test_NotFoundHandlerReturns404(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	handler.NotFoundHandler(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_NotFoundHandlerHandlesNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The function should panic")
		}
	}()
	handler.NotFoundHandler(nil)
}

func Test_NoMethodHandlerReturns405(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	handler.NoMethodHandler(c)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func Test_NoMethodHandlerHandlesNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The function should panic")
		}
	}()
	handler.NoMethodHandler(nil)
}
