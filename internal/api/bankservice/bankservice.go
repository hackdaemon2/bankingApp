package bankservice

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
	"github.com/google/uuid"
	"github.com/govalues/decimal"
	"github.com/mitchellh/mapstructure"
)

var headers = map[string]string{constants.ContentTypeHeader: constants.ContentTypeValue}

type IAccount interface {
	SetBalance(value decimal.Decimal)
	GetBalance() decimal.Decimal
	Deposit(amount decimal.Decimal) error
	Withdraw(amount decimal.Decimal) error
	IsInsufficientBalance(amount decimal.Decimal) bool
}

type IAccountRepository interface {
	SaveAccount(account *model.Account) error
	GetAccountByAccountNumber(number string) (*model.Account, error)
}

type IUserRepository interface {
	FindUserByUsername(username string) (model.User, error)
	GetUserByAccountNumber(accountNumber string) (*model.User, *model.Account, error)
}

type ITransactionRepository interface {
	FindTransaction(id uint) (*model.Transaction, error)
	FindTransactionByReference(reference string) (*model.Transaction, error)
	SaveTransaction(transaction *model.Transaction) error
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

	amount, err := decimal.NewFromFloat64(response["amount"].(float64))
	if err != nil {
		utility.HandleError(c, err, http.StatusInternalServerError, constants.ApplicationError)
		return
	}

	apiResponse := model.ResponseDTO{
		ThirdPartyTransactionDataDTO: model.ThirdPartyTransactionDataDTO{
			AccountID: response["account_id"].(string),
			Amount:    amount,
			Reference: response["reference"].(string),
		},
		PaymentReference: transaction.Reference,
	}

	c.JSON(http.StatusOK, utility.FormulateSuccessResponse(apiResponse))
}

func (b *BankTransferService) Transfer(c *gin.Context) {
	var t model.TransactionRequestDTO
	if err := c.BindJSON(&t); err != nil {
		utility.HandleError(c, err, http.StatusBadRequest, constants.InvalidJsonRequestErrorMsg)
		return
	}

	if err := b.validateTransferRequest(c, t); err != nil {
		return
	}

	transaction, err := b.TransactionRepository.FindTransactionByReference(t.Reference)
	if err != nil {
		utility.HandleError(c, err, http.StatusInternalServerError, constants.ApplicationError)
		return
	}

	if transaction.TransactionID != constants.Zero {
		utility.HandleError(c, nil, http.StatusOK, constants.NotUniqueReferenceMsg)
		return
	}

	user, account, err := b.UserRepository.GetUserByAccountNumber(t.AccountNumber)
	if err != nil {
		utility.HandleError(c, err, http.StatusInternalServerError, constants.ApplicationError)
		return
	}

	if user.UserID == constants.Zero || account.AccountID == constants.Zero {
		utility.HandleError(c, nil, http.StatusOK, constants.UserOrAccountNotFound)
		return
	}

	if t.TransactionPin != user.TransactionPin {
		utility.HandleError(c, nil, http.StatusOK, constants.IncorrectTransactionPin)
		return
	}

	if t.Type == model.DebitTransaction && account.IsInsufficientBalance(t.Amount) {
		utility.HandleError(c, nil, http.StatusOK, constants.InsufficientFunds)
		return
	}

	reference := uuid.New().String()

	url := fmt.Sprintf("%s/api/v1/third-party/payments", b.Config.ThirdPartyBaseUrl())
	accountID := strconv.Itoa(int(account.AccountID))
	request := &model.ThirdPartyTransactionDataDTO{
		AccountID: accountID,
		Amount:    t.Amount,
		Reference: reference,
	}

	response, statusCode, err := b.RestHttpClient.PostRequest(url, request, headers)
	if err != nil {
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
		slog.Error("map decode error", err) // nolint:govet
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

func (b *BankTransferService) isTransactionCreated(t transactionCreatedDTO) bool {
	if b.isSuccessfulTransaction(t.transactionRequest, t.account, t.context) {
		if err := b.AccountRepository.SaveAccount(t.account); err != nil {
			slog.Error("error in updating account balance", t.err) // nolint:govet
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
			slog.Error("error in save transaction", err) // nolint:govet
			utility.InternalServerError(t.context)
			return false
		}
		return true
	}
	return false
}

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

func (b *BankTransferService) handleDebit(amount decimal.Decimal, account *model.Account) error {
	return account.Withdraw(amount)
}

func (b *BankTransferService) handleCredit(amount decimal.Decimal, account *model.Account) error {
	return account.Deposit(amount)
}
