package bankservice_test //nolint:typecheck

import (
	"bankingApp/internal/api/bankservice"
	"bankingApp/internal/api/constants"
	"bankingApp/internal/model"
	"bankingApp/internal/utility"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/govalues/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type (
	MockUserRepository        struct{ mock.Mock }
	MockConfig                struct{ mock.Mock }
	MockTransactionRepository struct{ mock.Mock }
	MockAccountRepository     struct{ mock.Mock }
	MockRestHttpClient        struct{ mock.Mock }

	MockAccount struct {
		Balance model.Money
		mock.Mock
	}

	GinResponseWriter struct {
		gin.ResponseWriter
		Body []byte
	}
)

// mocks
func (a *MockConfig) ReadTimeout() uint32        { return a.Called().Get(0).(uint32) }
func (a *MockConfig) ServerPort() uint32         { return a.Called().Get(0).(uint32) }
func (a *MockConfig) ThirdPartyBaseUrl() string  { return a.Called().Get(0).(string) }
func (a *MockConfig) GinMode() string            { return a.Called().Get(0).(string) }
func (a *MockConfig) Username() string           { return a.Called().Get(0).(string) }
func (a *MockConfig) Password() string           { return a.Called().Get(0).(string) }
func (a *MockConfig) Host() string               { return a.Called().Get(0).(string) }
func (a *MockConfig) Port() int                  { return a.Called().Get(0).(int) }
func (a *MockConfig) DatabaseName() string       { return a.Called().Get(0).(string) }
func (a *MockConfig) MaximumOpenConnection() int { return a.Called().Get(0).(int) }
func (a *MockConfig) MaximumIdleConnection() int { return a.Called().Get(0).(int) }
func (a *MockConfig) MaximumIdleTime() int       { return a.Called().Get(0).(int) }
func (a *MockConfig) MaximumTime() int           { return a.Called().Get(0).(int) }
func (a *MockConfig) JwtSecret() string          { return a.Called().Get(0).(string) }

func (w *GinResponseWriter) Write(data []byte) (int, error) {
	w.Body = append(w.Body, data...)
	return w.ResponseWriter.Write(data)
}

func (u *MockUserRepository) GetUserAndAccountByAccountNumber(accountNumber string) (*model.User, *model.Account, error) {
	args := u.Called(accountNumber)
	return args.Get(0).(*model.User), args.Get(1).(*model.Account), args.Error(2)
}

func (u *MockUserRepository) FindUserByUsername(username string) (model.User, error) {
	args := u.Called(username)
	return args.Get(0).(model.User), args.Error(1)
}

func (m *MockTransactionRepository) FindTransaction(id uint) (*model.Transaction, error) {
	args := m.Called(id)
	return args.Get(0).(*model.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) SaveTransaction(transaction *model.Transaction) error {
	args := m.Called(transaction)
	return args.Error(0)
}

func (m *MockTransactionRepository) FindTransactionByReference(reference string) (*model.Transaction, error) {
	args := m.Called(reference)
	return args.Get(0).(*model.Transaction), args.Error(1)
}

func (a *MockAccountRepository) SaveAccount(account *model.Account) error {
	args := a.Called(account)
	return args.Error(0)
}

func (a *MockAccountRepository) GetAccountByAccountNumber(number string) (*model.Account, error) {
	args := a.Called(number)
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockRestHttpClient) GetRequest(
	url string,
	headers map[string]string) (map[string]interface{}, int, error) {
	args := m.Called(url, headers)
	return args.Get(0).(map[string]interface{}), args.Int(1), args.Error(2)
}

func (m *MockRestHttpClient) PostRequest(
	url string,
	request interface{},
	headers map[string]string) (map[string]interface{}, int, error) {
	args := m.Called(url, request, headers)
	return args.Get(0).(map[string]interface{}), args.Int(1), args.Error(2)
}

func (m *MockAccount) SetBalance(value model.Money) {
	m.Balance = value
}

func (m *MockAccount) GetBalance() model.Money {
	return m.Balance
}

func (m *MockAccount) Deposit(amount model.Money) error {
	if !amount.Decimal.IsPos() {
		return errors.New("cannot deposit negative amount")
	}
	newBalance, _ := m.Balance.Decimal.Add(amount.Decimal)
	m.Balance = model.Money{Decimal: newBalance}
	return nil
}

func (m *MockAccount) Withdraw(amount model.Money) error {
	if !amount.Decimal.IsPos() {
		return errors.New("cannot withdraw negative amount")
	}
	if m.Balance.Decimal.Cmp(amount.Decimal) == -1 {
		return errors.New("insufficient balance")
	}
	newBalance, _ := m.Balance.Decimal.Sub(amount.Decimal)
	m.Balance = model.Money{Decimal: newBalance}
	return nil
}

func (m *MockAccount) IsInsufficientBalance(amount model.Money) bool {
	return m.Balance.Decimal.Cmp(amount.Decimal) == -1
}

// tests

func Test_NewBankService(t *testing.T) {
	mockConfig, mockTransactionRepo, mockUserRepo, mockAccountRepo, mockRestClient, _ := setupMocks()
	bankService := bankservice.NewBankService(mockConfig, mockTransactionRepo, mockUserRepo, mockAccountRepo, mockRestClient)
	assert.NotNil(t, bankService)
	assert.Equal(t, mockConfig, bankService.Config)
	assert.Equal(t, mockTransactionRepo, bankService.TransactionRepository)
	assert.Equal(t, mockUserRepo, bankService.UserRepository)
	assert.Equal(t, mockAccountRepo, bankService.AccountRepository)
	assert.Equal(t, mockRestClient, bankService.RestHttpClient)
}

func Test_StatusQuery(t *testing.T) {
	val, _ := decimal.NewFromFloat64(100.00)
	amount := model.Money{Decimal: val}
	testCases := []struct {
		name             string
		reference        string
		mockTransaction  *model.Transaction
		mockAccount      *model.Account
		restResponse     map[string]interface{}
		restStatusCode   int
		restError        error
		dbError          error
		config           model.IAppConfiguration
		expectedResponse utility.APIResponse
		expectedStatus   int
		mockConfig       model.IAppConfiguration
	}{
		{
			name:             "Happy case",
			reference:        "289192938929293",
			mockTransaction:  getMockFoundTransaction(),
			restResponse:     getSuccessThirdPartyResponse(),
			restStatusCode:   http.StatusOK,
			restError:        nil,
			dbError:          nil,
			mockAccount:      getMockAccount(),
			expectedResponse: getExpectedResponse(amount),
			expectedStatus:   http.StatusOK,
			config:           getMockConfig(),
		},
		{
			name:             "Transaction not found",
			reference:        "289192938929293",
			mockTransaction:  getMockNotFoundTransaction(),
			expectedResponse: *utility.FormulateErrorResponse(constants.TransactionNotFound),
			expectedStatus:   http.StatusOK,
			config:           getMockConfig(),
		},
		{
			name:             "Find transaction throws DB error",
			reference:        "289192938929293",
			mockTransaction:  nil,
			dbError:          errors.New("something went wrong"),
			expectedResponse: *utility.FormulateErrorResponse("something went wrong"),
			expectedStatus:   http.StatusInternalServerError,
			config:           getMockConfig(),
		},
		{
			name:             "API call returns error",
			reference:        "289192938929293",
			mockTransaction:  getMockFoundTransaction(),
			dbError:          nil,
			expectedResponse: *utility.FormulateErrorResponse(constants.UnableToCompleteTransaction),
			expectedStatus:   http.StatusServiceUnavailable,
			config:           getMockConfig(),
			restStatusCode:   0,
			restError:        errors.New("unable to reach server"),
			mockAccount:      getMockAccount(),
		},
		{
			name:             "API call returns different HTTP Status Code",
			reference:        "289192938929293",
			mockTransaction:  getMockFoundTransaction(),
			dbError:          nil,
			expectedResponse: *utility.FormulateErrorResponse(constants.UnableToCompleteTransaction),
			expectedStatus:   http.StatusServiceUnavailable,
			config:           getMockConfig(),
			restStatusCode:   http.StatusGatewayTimeout,
			restError:        nil,
			mockAccount:      getMockAccount(),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockConfig, mockTransactionRepo, mockUserRepo, mockAccountRepo, mockRestClient, _ := setupMocks()
			bankService := createBankService(mockConfig, mockTransactionRepo, mockUserRepo, mockAccountRepo, mockRestClient)

			gin.SetMode(gin.TestMode)

			mockConfig.On("ThirdPartyBaseUrl").Return(mock.Anything)

			mockTransactionRepo.
				On("FindTransactionByReference", tt.reference).Return(tt.mockTransaction, tt.dbError)

			mockRestClient.
				On("GetRequest", mock.Anything, mock.Anything).Return(tt.restResponse, tt.restStatusCode, tt.restError)

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			w := &GinResponseWriter{ResponseWriter: ctx.Writer}
			ctx.Params = append(ctx.Params, gin.Param{Key: "ref", Value: tt.reference})
			ctx.Writer = w

			bankService.StatusQuery(ctx)

			var returnedResponse utility.APIResponse
			jsonErr := json.Unmarshal(w.Body, &returnedResponse)
			if jsonErr != nil {
				t.Logf("Error: %v", jsonErr)
				t.Fatalf("Error creating test context: %v", jsonErr)
				return
			}

			assert.Equal(t, tt.expectedStatus, ctx.Writer.Status())
			assert.Equal(t, tt.expectedResponse.Success, returnedResponse.Success)
			assert.Equal(t, tt.expectedResponse.Data, returnedResponse.Data)
			assert.Equal(t, tt.expectedResponse.Message, returnedResponse.Message)
		})
	}
}

func Test_Transfer(t *testing.T) {
	val, _ := decimal.NewFromFloat64(100.00)
	amount := model.Money{Decimal: val}
	valInsufficientFunds, _ := decimal.NewFromFloat64(1_000_000_000.00)
	amountInsufficientFunds := model.Money{Decimal: valInsufficientFunds}
	testCases := []struct {
		name                      string
		mockTransaction           *model.Transaction
		mockAccount               *model.Account
		restResponse              map[string]interface{}
		restStatusCode            int
		restError                 error
		dbError                   error
		mockUser                  *model.User
		config                    model.IAppConfiguration
		expectedResponse          utility.APIResponse
		expectedStatus            int
		mockConfig                model.IAppConfiguration
		requestBody               []byte
		insufficientBalance       bool
		transactionType           model.TransactionType
		transactionTypeSuccessful bool
	}{
		{
			name:             "successful debit test case",
			mockTransaction:  getMockNotFoundTransaction(),
			restResponse:     getSuccessThirdPartyResponse(),
			restStatusCode:   http.StatusOK,
			restError:        nil,
			dbError:          nil,
			mockAccount:      getMockAccount(),
			expectedResponse: getExpectedResponse(amount),
			expectedStatus:   http.StatusOK,
			config:           getMockConfig(),
			requestBody: getTransactionRequest(
				"1234567890",
				"johndoe",
				"1234",
				"289192938929293",
				model.DebitTransaction,
				amount),
			mockUser:                  getMockUser(),
			insufficientBalance:       false,
			transactionTypeSuccessful: true,
			transactionType:           model.DebitTransaction,
		},
		{
			name:             "successful credit test case",
			mockTransaction:  getMockNotFoundTransaction(),
			restResponse:     getSuccessThirdPartyResponse(),
			restStatusCode:   http.StatusOK,
			restError:        nil,
			dbError:          nil,
			mockAccount:      getMockAccount(),
			expectedResponse: getExpectedResponse(amount),
			expectedStatus:   http.StatusOK,
			config:           getMockConfig(),
			requestBody: getTransactionRequest(
				"1234567890",
				"johndoe",
				"1234",
				"289192938929293",
				model.CreditTransaction,
				amount),
			mockUser:                  getMockUser(),
			insufficientBalance:       false,
			transactionTypeSuccessful: true,
			transactionType:           model.DebitTransaction,
		},
		{
			name:             "bad request invalid type test case",
			expectedResponse: getExpectedBadRequestResponse(),
			expectedStatus:   http.StatusBadRequest,
			config:           getMockConfig(),
			requestBody: getTransactionRequest(
				"1234567890",
				"johndoe",
				"1234",
				"289192938929293",
				"flier",
				amount),
		},
		{
			name:             "invalid PIN test case",
			expectedResponse: getErrorResponse(constants.IncorrectTransactionPin),
			expectedStatus:   http.StatusOK,
			config:           getMockConfig(),
			mockUser:         getMockUser(),
			mockTransaction:  getMockNotFoundTransaction(),
			mockAccount:      getMockAccount(),
			requestBody: getTransactionRequest(
				"1234567890",
				"johndoe",
				"1345",
				"289192938929293",
				"debit",
				amount),
		},
		{
			name:             "invalid PIN length test case",
			expectedResponse: getErrorResponse(constants.BadRequestMessage),
			expectedStatus:   http.StatusBadRequest,
			config:           getMockConfig(),
			requestBody: getTransactionRequest(
				"1234567890",
				"johndoe",
				"12345",
				"289192938929293",
				"debit",
				amount),
		},
		{
			name:             "insufficient funds for debit test case",
			mockTransaction:  getMockNotFoundTransaction(),
			restStatusCode:   http.StatusOK,
			restError:        nil,
			dbError:          nil,
			mockAccount:      getMockAccount(),
			expectedResponse: getErrorResponse(constants.InsufficientFunds),
			expectedStatus:   http.StatusOK,
			config:           getMockConfig(),
			requestBody: getTransactionRequest(
				"1234567890",
				"johndoe",
				"1234",
				"289192938929293",
				model.DebitTransaction,
				amountInsufficientFunds),
			mockUser:                  getMockUser(),
			insufficientBalance:       true,
			transactionTypeSuccessful: false,
			transactionType:           model.DebitTransaction,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockConfig, mockTransactionRepo, mockUserRepo, mockAccountRepo, mockRestClient, mockAccount := setupMocks()
			bankService := createBankService(mockConfig, mockTransactionRepo, mockUserRepo, mockAccountRepo, mockRestClient)

			gin.SetMode(gin.TestMode)

			mockConfig.
				On("ThirdPartyBaseUrl").
				Return(mock.Anything)

			mockTransactionRepo.
				On("FindTransactionByReference", mock.Anything).Return(tt.mockTransaction, tt.dbError)

			mockTransactionRepo.
				On("SaveTransaction", mock.Anything).Return(tt.dbError)

			mockAccountRepo.
				On("SaveAccount", mock.Anything).Return(tt.dbError)

			mockRestClient.
				On("PostRequest", mock.Anything, mock.Anything, mock.Anything).
				Return(tt.restResponse, tt.restStatusCode, tt.restError)

			mockUserRepo.
				On("GetUserAndAccountByAccountNumber", mock.Anything).Return(tt.mockUser, tt.mockAccount, nil)

			mockAccount.On("IsInsufficientBalance", mock.Anything).Return(tt.insufficientBalance)

			var methodName string
			if tt.transactionType == model.DebitTransaction {
				methodName = "Withdraw"
			} else if tt.transactionType == model.CreditTransaction {
				methodName = "Deposit"
			}

			mockAccount.On(methodName, mock.Anything).Return(tt.transactionTypeSuccessful)

			req, err := http.NewRequest("POST", "/api/v1/bank/fund-transfer", bytes.NewBuffer(tt.requestBody))
			if err != nil {
				t.Logf("Error creating request: %v", err)
				t.Fatalf("Error creating request context: %v", err)
				return
			}

			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = req

			w := &GinResponseWriter{ResponseWriter: context.Writer}
			context.Writer = w

			bankService.Transfer(context)

			var returnedResponse utility.APIResponse
			jsonErr := json.Unmarshal(w.Body, &returnedResponse)
			if jsonErr != nil {
				t.Logf("Error: %v", jsonErr)
				t.Fatalf("Error creating test context: %v", jsonErr)
				return
			}

			t.Logf("response => %s", string(w.Body))

			assert.Equal(t, tt.expectedStatus, context.Writer.Status())
			assert.Equal(t, tt.expectedResponse.Success, returnedResponse.Success)
			assert.Equal(t, tt.expectedResponse.Data, returnedResponse.Data)
			assert.Equal(t, tt.expectedResponse.Message, returnedResponse.Message)
		})
	}
}

// helper functions

func getSuccessThirdPartyResponse() map[string]interface{} {
	return map[string]interface{}{
		"amount":     100.0,
		"account_id": "1",
		"reference":  "aeda214f-513b-4e80-8e9d-2e513054a148",
	}
}

func getMockConfig() model.IAppConfiguration {
	return &MockConfig{}
}

func getMockFoundTransaction() *model.Transaction {
	return &model.Transaction{
		TransactionID: 1,
		Reference:     "289192938929293",
	}
}

func getMockNotFoundTransaction() *model.Transaction {
	return &model.Transaction{
		TransactionID: 0,
		Reference:     "",
	}
}

func getExpectedResponse(amount model.Money) utility.APIResponse {
	return utility.APIResponse{
		Data: &model.ResponseDTO{
			ThirdPartyTransactionDataDTO: model.ThirdPartyTransactionDataDTO{
				AccountID: "1",
				Amount:    &amount,
				Reference: "aeda214f-513b-4e80-8e9d-2e513054a148",
			},
			PaymentReference: "289192938929293",
		},
		Success: true,
		Message: constants.SuccessfulTransactionMsg,
	}
}

func getExpectedBadRequestResponse() utility.APIResponse {
	return utility.APIResponse{
		Success: false,
		Message: constants.BadRequestMessage,
		Errors:  make(map[string]string),
	}
}

func getErrorResponse(message string) utility.APIResponse {
	return utility.APIResponse{
		Success: false,
		Message: message,
		Errors:  make(map[string]string),
	}
}

func getMockAccount() *model.Account {
	balance, _ := decimal.NewFromFloat64(100000)
	return &model.Account{
		AccountID:     1,
		AccountNumber: "1234567890",
		UserID:        1,
		Balance:       model.Money{Decimal: balance},
	}
}

func getMockUser() *model.User {
	return &model.User{
		UserID:         1,
		Username:       "1234567890",
		TransactionPin: "1234",
	}
}

func getTransactionRequest(account, username, pin, reference string,
	transactionType model.TransactionType, amount model.Money) []byte {
	requestBody, _ := json.Marshal(model.TransactionRequestDTO{
		TransactionDataDTO: model.TransactionDataDTO{
			AccountNumber:  account,
			Username:       username,
			TransactionPin: pin,
			Reference:      reference,
			Amount:         amount,
			Type:           transactionType,
		},
	})
	fmt.Println("AMOUNT => " + string(requestBody))
	return requestBody
}

func setupMocks() (*MockConfig, *MockTransactionRepository, *MockUserRepository,
	*MockAccountRepository, *MockRestHttpClient, *MockAccount) {
	return new(MockConfig),
		new(MockTransactionRepository),
		new(MockUserRepository),
		new(MockAccountRepository),
		new(MockRestHttpClient),
		new(MockAccount)
}

func createBankService(config *MockConfig, transactionRepo *MockTransactionRepository,
	userRepo *MockUserRepository, accountRepo *MockAccountRepository,
	restClient *MockRestHttpClient) *bankservice.BankTransferService {
	return &bankservice.BankTransferService{
		Config:                config,
		TransactionRepository: transactionRepo,
		RestHttpClient:        restClient,
		UserRepository:        userRepo,
		AccountRepository:     accountRepo,
	}
}
