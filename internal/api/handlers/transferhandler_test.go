package handler_test // nolint:typecheck

import (
	handler "bankingApp/internal/api/handlers"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type (
	MockBankTransferService struct{ mock.Mock }

	GinResponseWriter struct {
		gin.ResponseWriter
		Body []byte
	}
)

// mocks

func (g *GinResponseWriter) Write(data []byte) (int, error) {
	g.Body = append(g.Body, data...)
	return g.ResponseWriter.Write(data)
}

func (m *MockBankTransferService) StatusQuery(context *gin.Context) {
	m.Called(context).Get(0)
}

func (m *MockBankTransferService) Transfer(context *gin.Context) {
	m.Called(context).Get(0)
}

// tests

func Test_NewBankTransferHandler(t *testing.T) {
	mockBankTransferService := new(MockBankTransferService)
	transferHandler := handler.NewBankTransferHandler(mockBankTransferService)
	assert.NotNil(t, transferHandler)
	assert.Equal(t, mockBankTransferService, transferHandler.BankTransferService)
}

func Test_StatusQuery(t *testing.T) {
	mockBankTransferService := new(MockBankTransferService)
	transferHandler := handler.NewBankTransferHandler(mockBankTransferService)
	testCases := []struct {
		name        string
		handlerFunc func(*gin.Context)
	}{
		{
			name:        "StatusQuery test case",
			handlerFunc: transferHandler.StatusQuery,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			gin.SetMode(gin.TestMode)
			mockBankTransferService.On("StatusQuery", ctx).Return(mock.Anything)
			w := &GinResponseWriter{ResponseWriter: ctx.Writer}
			ctx.Writer = w
			tt.handlerFunc(ctx)
			mockBankTransferService.AssertCalled(t, "StatusQuery", ctx)
		})
	}
}

func Test_Transfer(t *testing.T) {
	mockBankTransferService := new(MockBankTransferService)
	transferHandler := handler.NewBankTransferHandler(mockBankTransferService)
	testCases := []struct {
		name        string
		handlerFunc func(*gin.Context)
	}{
		{
			name:        "Transfer test case",
			handlerFunc: transferHandler.Transfer,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			gin.SetMode(gin.TestMode)
			mockBankTransferService.On("Transfer", ctx).Return(mock.Anything)
			w := &GinResponseWriter{ResponseWriter: ctx.Writer}
			ctx.Writer = w
			tt.handlerFunc(ctx)
			mockBankTransferService.AssertCalled(t, "Transfer", ctx)
		})
	}
}
