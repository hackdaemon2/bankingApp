package bankservice_test

import (
	"bankingApp/internal/api/bankservice"
	"bankingApp/internal/api/constants"
	"bankingApp/internal/model"
	"bankingApp/internal/utility"
	"bytes"
	"encoding/json"
	"errors"
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
		Balance decimal.Decimal
		mock.Mock
	}

	GinResponseWriter struct {
		gin.ResponseWriter
		Body []byte
	}
)

// mock config
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

func (u *MockUserRepository) GetUserByAccountNumber(accountNumber string) (*model.User, *model.Account, error) {
	args := u.Called(accountNumber) // nolint:typecheck
	return args.Get(0).(*model.User), args.Get(1).(*model.Account), args.Error(2)
}

func (u *MockUserRepository) FindUserByUsername(username string) (model.User, error) {
	args := u.Called(username) // nolint:typecheck
	return args.Get(0).(model.User), args.Error(1)
}

func (m *MockTransactionRepository) FindTransaction(id uint) (*model.Transaction, error) {
	args := m.Called(id) // nolint:typecheck
	return args.Get(0).(*model.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) SaveTransaction(transaction *model.Transaction) error {
	args := m.Called(transaction) // nolint:typecheck
	return args.Error(0)
}

func (m *MockTransactionRepository) FindTransactionByReference(reference string) (*model.Transaction, error) {
	args := m.Called(reference) // nolint:typecheck
	return args.Get(0).(*model.Transaction), args.Error(1)
}

func (a *MockAccountRepository) SaveAccount(account *model.Account) error {
	args := a.Called(account) // nolint:typecheck
	return args.Error(0)
}

func (a *MockAccountRepository) GetAccountByAccountNumber(number string) (*model.Account, error) {
	args := a.Called(number) // nolint:typecheck
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockRestHttpClient) GetRequest(
	url string,
	headers map[string]string) (map[string]interface{}, int, error) {
	args := m.Called(url, headers) // nolint:typecheck
	return args.Get(0).(map[string]interface{}), args.Int(1), args.Error(2)
}

func (m *MockRestHttpClient) PostRequest(
	url string,
	request interface{},
	headers map[string]string) (map[string]interface{}, int, error) {
	args := m.Called(url, request, headers) // nolint:typecheck
	return args.Get(0).(map[string]interface{}), args.Int(1), args.Error(2)
}

func (m *MockAccount) SetBalance(value decimal.Decimal) {
	m.Balance = value
}

func (m *MockAccount) GetBalance() decimal.Decimal {
	return m.Balance
}

func (m *MockAccount) Deposit(amount decimal.Decimal) error {
	if !amount.IsPos() {
		return errors.New("cannot deposit negative amount")
	}
	m.Balance, _ = m.Balance.Add(amount)
	return nil
}

func (m *MockAccount) Withdraw(amount decimal.Decimal) error {
	if !amount.IsPos() {
		return errors.New("cannot withdraw negative amount")
	}
	if m.Balance.Cmp(amount) == -1 {
		return errors.New("insufficient balance")
	}
	m.Balance, _ = m.Balance.Sub(amount)
	return nil
}

func (m *MockAccount) IsInsufficientBalance(amount decimal.Decimal) bool {
	return m.Balance.Cmp(amount) == -1
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
	amount, _ := decimal.NewFromFloat64(100.00)
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
				On("FindTransactionByReference", tt.reference).Return(tt.mockTransaction, tt.dbError) // nolint:typecheck

			mockRestClient.
				On("GetRequest", mock.Anything, mock.Anything).Return(tt.restResponse, tt.restStatusCode, tt.restError) // nolint:typecheck

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
	amount, _ := decimal.NewFromFloat64(100.00)
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
				On("FindTransactionByReference", mock.Anything).Return(tt.mockTransaction, tt.dbError) // nolint:typecheck

			mockTransactionRepo.
				On("SaveTransaction", mock.Anything).Return(tt.dbError) // nolint:typecheck

			mockAccountRepo.
				On("SaveAccount", mock.Anything).Return(tt.dbError) // nolint:typecheck

			mockRestClient.
				On("PostRequest", mock.Anything, mock.Anything, mock.Anything).
				Return(tt.restResponse, tt.restStatusCode, tt.restError) // nolint:typecheck

			mockUserRepo.
				On("GetUserByAccountNumber", mock.Anything).Return(tt.mockUser, tt.mockAccount, nil) // nolint:typecheck

			mockAccount.On("IsInsufficientBalance", mock.Anything).Return(tt.insufficientBalance) // nolint:typecheck

			if tt.transactionType == model.DebitTransaction {
				mockAccount.On("Withdraw", mock.Anything).Return(tt.transactionTypeSuccessful) // nolint:typecheck
			} else if tt.transactionType == model.CreditTransaction {
				mockAccount.On("Deposit", mock.Anything).Return(tt.transactionTypeSuccessful) // nolint:typecheck
			}

			req, err := http.NewRequest("POST", "/api/v1/bank/fund-transfer", bytes.NewBuffer(tt.requestBody))
			if err != nil {
				t.Logf("Error creating request: %v", err)
				t.Fatalf("Error creating request context: %v", err)
				return
			}

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = req

			w := &GinResponseWriter{ResponseWriter: c.Writer}
			c.Writer = w

			bankService.Transfer(c)

			var returnedResponse utility.APIResponse
			jsonErr := json.Unmarshal(w.Body, &returnedResponse)
			if jsonErr != nil {
				t.Logf("Error: %v", jsonErr)
				t.Fatalf("Error creating test context: %v", jsonErr)
				return
			}

			t.Logf("response => %s", string(w.Body))

			assert.Equal(t, tt.expectedStatus, c.Writer.Status())
			assert.Equal(t, tt.expectedResponse.Success, returnedResponse.Success)
			assert.Equal(t, tt.expectedResponse.Data, returnedResponse.Data)
			assert.Equal(t, tt.expectedResponse.Message, returnedResponse.Message)
		})
	}
}

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

func getExpectedResponse(amount decimal.Decimal) utility.APIResponse {
	return utility.APIResponse{
		Data: model.ResponseDTO{
			ThirdPartyTransactionDataDTO: model.ThirdPartyTransactionDataDTO{
				AccountID: "1",
				Amount:    amount,
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
		Balance:       balance,
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
	transactionType model.TransactionType, amount decimal.Decimal) []byte {
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
	return requestBody
}
