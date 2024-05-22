package bankservice // nolint:govet

import (
	"bankingApp/internal/api/constants"
	"bankingApp/internal/model"
	"bankingApp/internal/utility"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
)

var headers = map[string]string{constants.ContentTypeHeader: constants.ContentTypeValue}

type IAccount interface {
	SetBalance(value model.BigDecimal)
	GetBalance() model.BigDecimal
	Deposit(amount model.BigDecimal) error
	Withdraw(amount model.BigDecimal) error
	IsInsufficientBalance(amount model.BigDecimal) bool
}

type IAccountRepository interface {
	UpdateAccount(account *model.Account) error
	GetAccountByAccountNumber(number string) (*model.Account, error)
}

type IUserRepository interface {
	FindUserByUsername(username string) (model.User, error)
	GetUserAndAccountByAccountNumber(accountNumber string) (*model.User, *model.Account, error)
}

type ITransactionRepository interface {
	FindTransaction(id uint) (*model.Transaction, error)
	FindTransactionByReference(reference string) (*model.Transaction, error)
	SaveTransaction(transaction *model.Transaction) error
	GetLastInsertID() (uint, error)
}

type IRestHttpClient interface {
	GetRequest(url string, headers map[string]string) (map[string]interface{}, int, error)
	PostRequest(url string, request interface{}, headers map[string]string) (map[string]interface{}, int, error)
}

type BankTransferService struct {
	Config                model.IAppConfiguration
	TransactionRepository ITransactionRepository
	UserRepository        IUserRepository
	AccountRepository     IAccountRepository
	RestHttpClient        IRestHttpClient
}

type transactionCreatedDTO struct {
	context            *gin.Context
	account            *model.Account
	transactionRequest model.TransactionRequestDTO
	statusCode         int
	reference          string
	err                error
}

// NewBankService initializes a new BankTransferService with the provided dependencies.
// It returns a pointer to the created BankTransferService instance.
func NewBankService(
	config model.IAppConfiguration,
	transactionRepo ITransactionRepository,
	userRepo IUserRepository,
	accountRepo IAccountRepository,
	restClient IRestHttpClient) *BankTransferService {
	return &BankTransferService{
		Config:                config,
		TransactionRepository: transactionRepo,
		UserRepository:        userRepo,
		AccountRepository:     accountRepo,
		RestHttpClient:        restClient,
	}
}

// StatusQuery handles the status query endpoint for checking transaction status.
// It retrieves transaction details, makes a request to a third-party service, and returns the response.
func (b *BankTransferService) StatusQuery(c *gin.Context) {
	reference := c.Param("ref")

	transaction, err := b.TransactionRepository.FindTransactionByReference(reference)
	if err != nil {
		utility.HandleError(c, err, http.StatusInternalServerError, err.Error())
		return
	}

	if transaction.TransactionID == constants.Zero {
		utility.HandleError(c, nil, http.StatusOK, constants.TransactionNotFound)
		return
	}

	url := fmt.Sprintf("%s/api/v1/third-party/payments/%s/get", b.Config.ThirdPartyBaseUrl(), transaction.Reference)

	response, statusCode, err := b.RestHttpClient.GetRequest(url, headers)
	if err != nil {
		utility.HandleError(c, err, http.StatusServiceUnavailable, constants.UnableToCompleteTransaction)
		return
	}

	if statusCode != http.StatusOK {
		utility.HandleError(c, nil, http.StatusServiceUnavailable, constants.UnableToCompleteTransaction)
		return
	}

	amount := transaction.Amount.Decimal
	apiResponse := model.ResponseDTO{
		ThirdPartyTransactionDataDTO: model.ThirdPartyTransactionDataDTO{
			AccountID: response["account_id"].(string),
			Amount:    &model.BigDecimal{Decimal: amount},
			Reference: transaction.Reference,
		},
		PaymentReference: transaction.PaymentReference,
	}

	c.JSON(http.StatusOK, utility.FormulateSuccessResponse(apiResponse))
}

// Transfer handles the transfer endpoint for initiating a transaction.
// It validates the transfer request, processes the transaction, and saves the transaction details.
func (b *BankTransferService) Transfer(c *gin.Context) {
	var t model.TransactionRequestDTO
	if err := c.BindJSON(&t); err != nil {
		utility.HandleError(c, err, http.StatusBadRequest, constants.InvalidJsonRequestErrorMsg)
		return
	}

	if err := b.validateTransferRequest(c, t); err != nil {
		return
	}

	err, account, complete := b.processValidation(c, t)
	if complete {
		return
	}

	lastInsertID, done := b.getLastInsertID(c, err)
	if done {
		return
	}

	reference := fmt.Sprintf("ref%d", lastInsertID+1)

	url := fmt.Sprintf("%s/api/v1/third-party/payments", b.Config.ThirdPartyBaseUrl())
	accountID := strconv.Itoa(int(account.AccountID))
	request := &model.ThirdPartyTransactionDataDTO{
		AccountID: accountID,
		Amount:    &t.Amount,
		Reference: reference,
	}

	response, statusCode, err := b.RestHttpClient.PostRequest(url, request, headers)
	if err != nil || statusCode != http.StatusOK {
		utility.HandleError(c, err, http.StatusInternalServerError, constants.ApplicationError)
		return
	}

	var thirdPartyResponse model.ThirdPartyTransactionDataDTO

	config := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			utility.Float64ToDecimalHookFunc,
		),
		Result: &thirdPartyResponse,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		slog.Error("map decode error", err)
	}

	if err = decoder.Decode(response); err != nil {
		utility.HandleError(c, err, http.StatusInternalServerError, constants.ApplicationError)
		return
	}

	apiResponse := model.ResponseDTO{
		ThirdPartyTransactionDataDTO: thirdPartyResponse,
		PaymentReference:             t.Reference,
	}

	transactionCreatedDto := transactionCreatedDTO{
		context:            c,
		account:            account,
		transactionRequest: t,
		reference:          reference,
		statusCode:         statusCode,
		err:                err,
	}

	if !b.isTransactionCreated(transactionCreatedDto) {
		return
	}

	c.JSON(http.StatusOK, utility.FormulateSuccessResponse(apiResponse))
}

func (b *BankTransferService) processValidation(c *gin.Context, t model.TransactionRequestDTO) (error, *model.Account, bool) {
	transaction, err := b.TransactionRepository.FindTransactionByReference(t.Reference)
	if err != nil {
		utility.HandleError(c, err, http.StatusInternalServerError, constants.ApplicationError)
		return nil, nil, true
	}

	if transaction.TransactionID != constants.Zero {
		utility.HandleError(c, nil, http.StatusOK, constants.NotUniqueReferenceMsg)
		return nil, nil, true
	}

	user, account, err := b.UserRepository.GetUserAndAccountByAccountNumber(t.AccountNumber)
	if err != nil {
		utility.HandleError(c, err, http.StatusInternalServerError, constants.ApplicationError)
		return nil, nil, true
	}

	if user.UserID == constants.Zero || account.AccountID == constants.Zero {
		utility.HandleError(c, nil, http.StatusOK, constants.UserOrAccountNotFound)
		return nil, nil, true
	}

	if t.TransactionPin != user.TransactionPin {
		utility.HandleError(c, nil, http.StatusOK, constants.IncorrectTransactionPin)
		return nil, nil, true
	}

	if t.Type == model.DebitTransaction && account.IsInsufficientBalance(t.Amount) {
		utility.HandleError(c, nil, http.StatusOK, constants.InsufficientFunds)
		return nil, nil, true
	}

	return err, account, false
}

func (b *BankTransferService) getLastInsertID(c *gin.Context, err error) (uint, bool) {
	lastInsertID, err := b.TransactionRepository.GetLastInsertID()
	if err != nil {
		utility.HandleError(c, err, http.StatusInternalServerError, constants.ApplicationError)
		return 0, true
	}
	return lastInsertID, false
}

// validateTransferRequest validates the transaction request data and handles any validation errors.
func (b *BankTransferService) validateTransferRequest(c *gin.Context, t model.TransactionRequestDTO) error {
	if errorMap, vErr := utility.ValidateRequest(t); len(errorMap) != constants.Zero || vErr != nil {
		if vErr != nil {
			utility.HandleError(c, vErr, http.StatusInternalServerError, constants.ApplicationError)
			return vErr
		}
		utility.HandleValidationErrors(c, errorMap)
		return fmt.Errorf("validation error")
	}
	return nil
}

// isTransactionCreated checks if a transaction was successfully created and updates account and transaction details accordingly.
func (b *BankTransferService) isTransactionCreated(t transactionCreatedDTO) bool {
	if b.isSuccessfulTransaction(t.transactionRequest, t.account, t.context) {
		if err := b.AccountRepository.UpdateAccount(t.account); err != nil {
			slog.Error("error in updating account balance", t.err)
			utility.InternalServerError(t.context)
			return false
		}

		transaction := &model.Transaction{
			AccountID:        t.account.AccountID,
			Amount:           t.transactionRequest.Amount,
			Type:             t.transactionRequest.Type,
			Success:          t.statusCode == http.StatusOK,
			Reference:        t.reference,
			PaymentReference: t.transactionRequest.Reference,
			TransactionTime:  time.Now(),
			TimestampData: model.TimestampData{
				CreatedAt: time.Now(),
			},
		}

		if err := b.TransactionRepository.SaveTransaction(transaction); err != nil {
			slog.Error("error in save transaction", err)
			utility.InternalServerError(t.context)
			return false
		}
		return true
	}
	return false
}

// isSuccessfulTransaction processes the transaction based on its type (debit or credit) and updates the account balance.
func (b *BankTransferService) isSuccessfulTransaction(t model.TransactionRequestDTO, a *model.Account, ctx *gin.Context) bool {
	switch t.Type {
	case model.DebitTransaction:
		if err := b.handleDebit(t.Amount, a); err != nil {
			slog.Error("debit transaction failed: ", err) //nolint:govet
			utility.InternalServerError(ctx)
			return false
		}
	case model.CreditTransaction:
		if err := b.handleCredit(t.Amount, a); err != nil {
			slog.Error("credit transaction failed: ", err) //nolint:govet
			utility.InternalServerError(ctx)
			return false
		}
	default:
		slog.Error("invalid transaction type")
		utility.InternalServerError(ctx)
		return false
	}
	return true
}

// handleDebit updates the account balance by withdrawing the specified amount for a debit transaction.
func (b *BankTransferService) handleDebit(amount model.BigDecimal, account *model.Account) error {
	return account.Withdraw(amount)
}

// handleCredit updates the account balance by depositing the specified amount for a credit transaction.
func (b *BankTransferService) handleCredit(amount model.BigDecimal, account *model.Account) error {
	return account.Deposit(amount)
}
