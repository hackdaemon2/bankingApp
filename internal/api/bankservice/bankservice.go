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
	context             *gin.Context
	account             *model.Account
	transactionRequest  model.TransactionRequestDTO
	statusCode          int
	bankTransferService *BankTransferService
	paymentReference    string
	err                 error
}

func New(config model.IAppConfiguration, transactionRepository ITransactionRepository,
	userRepository IUserRepository, accountRepository IAccountRepository,
	restHttpClient IRestHttpClient) *BankTransferService {
	return &BankTransferService{
		Config:                config,
		TransactionRepository: transactionRepository,
		UserRepository:        userRepository,
		AccountRepository:     accountRepository,
		RestHttpClient:        restHttpClient,
	}
}

func (b *BankTransferService) StatusQuery(context *gin.Context) {
	reference := context.Param("ref")

	transaction, err := b.TransactionRepository.FindTransactionByReference(reference)
	if err != nil {
		utility.HandleError(context, err, http.StatusInternalServerError, err.Error())
		return
	}

	if transaction.TransactionID == constants.Zero {
		utility.HandleError(context, nil, http.StatusOK, constants.TransactionNotFound)
		return
	}

	url := fmt.Sprintf("%s/api/v1/third-party/payments/%s/get", b.Config.ThirdPartyBaseUrl(), reference)

	response, statusCode, httpErr := b.RestHttpClient.GetRequest(url, headers)
	if httpErr != nil {
		utility.HandleError(context, httpErr, http.StatusInternalServerError, constants.ApplicationError)
		return
	}

	if statusCode != http.StatusOK {
		utility.HandleError(context, nil, statusCode, constants.UnableToCompleteTransaction)
		return
	}

	amount, convertErr := decimal.NewFromFloat64(response["amount"].(float64))
	if convertErr != nil {
		utility.HandleError(context, convertErr, http.StatusInternalServerError, constants.ApplicationError)
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

	context.JSON(http.StatusOK, utility.FormulateSuccessResponse(apiResponse))
}

func (b *BankTransferService) validateRequest(context *gin.Context, t model.TransactionRequestDTO) error {
	if errorMap, vErr := utility.ValidateRequest(t); len(errorMap) != constants.Zero || vErr != nil {
		if vErr != nil {
			utility.HandleError(context, vErr, http.StatusInternalServerError, constants.ApplicationError)
			return vErr
		}
		utility.HandleValidationErrors(context, errorMap)
		return fmt.Errorf("validation error")
	}
	return nil
}

func (b *BankTransferService) Transfer(context *gin.Context) {
	var t model.TransactionRequestDTO
	if err := context.BindJSON(&t); err != nil {
		utility.HandleError(context, err, http.StatusBadRequest, constants.InvalidJsonRequestErrorMsg)
		return
	}

	if err := b.validateRequest(context, t); err != nil {
		return
	}

	transaction, err := b.TransactionRepository.FindTransactionByReference(t.Reference)
	if err != nil {
		utility.HandleError(context, err, http.StatusInternalServerError, constants.ApplicationError)
		return
	}

	if transaction.TransactionID != constants.Zero {
		utility.HandleError(context, nil, http.StatusOK, constants.NotUniqueReferenceMsg)
		return
	}

	user, account, err := b.UserRepository.GetUserByAccountNumber(t.AccountNumber)
	if err != nil {
		utility.HandleError(context, err, http.StatusInternalServerError, constants.ApplicationError)
		return
	}

	if user.UserID == constants.Zero || account.AccountID == constants.Zero {
		utility.HandleError(context, nil, http.StatusOK, constants.UserOrAccountNotFound)
		return
	}

	if t.TransactionPin != user.TransactionPin {
		utility.HandleError(context, nil, http.StatusOK, constants.IncorrectTransactionPin)
		return
	}

	if t.Type == model.DebitTransaction && account.IsInsufficientBalance(t.Amount) {
		utility.HandleError(context, nil, http.StatusOK, constants.InsufficientFunds)
		return
	}

	paymentReference := uuid.New().String()

	url := fmt.Sprintf("%s/api/v1/third-party/payments", b.Config.ThirdPartyBaseUrl())
	accountID := strconv.Itoa(int(account.AccountID))
	request := &model.ThirdPartyTransactionDataDTO{
		AccountID: accountID,
		Amount:    t.Amount,
		Reference: paymentReference,
	}

	response, statusCode, httpErr := b.RestHttpClient.PostRequest(url, request, headers)
	if httpErr != nil {
		utility.HandleError(context, httpErr, http.StatusInternalServerError, constants.ApplicationError)
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
		slog.Error("Error creating decoder: %v", err)
	}

	err = decoder.Decode(response)
	if err != nil {
		utility.HandleError(context, err, http.StatusInternalServerError, constants.ApplicationError)
		return
	}

	apiResponse := model.ResponseDTO{
		ThirdPartyTransactionDataDTO: thirdPartyResponse,
		PaymentReference:             t.Reference,
	}

	transactionCreatedDto := transactionCreatedDTO{
		context:            context,
		account:            account,
		transactionRequest: t,
		paymentReference:   paymentReference,
		statusCode:         statusCode,
		err:                err,
	}

	if !b.isTransactionCreated(transactionCreatedDto) {
		return
	}

	context.JSON(http.StatusOK, utility.FormulateSuccessResponse(apiResponse))
}

func (b *BankTransferService) isTransactionCreated(t transactionCreatedDTO) bool {
	isNoError := b.isSuccessfulTransaction(t.transactionRequest, t.account, t.context)
	if isNoError {
		saveErr := b.AccountRepository.SaveAccount(t.account)
		if saveErr != nil {
			slog.Error(t.err.Error())
			utility.InternalServerError(t.context)
			return false
		}

		timestamp := time.Now()
		transaction := &model.Transaction{
			AccountID:        t.account.AccountID,
			Amount:           t.transactionRequest.Amount,
			Type:             t.transactionRequest.Type,
			Success:          t.statusCode == http.StatusOK,
			Reference:        t.transactionRequest.Reference,
			PaymentReference: t.paymentReference,
			TransactionTime:  timestamp,
			TimestampData: model.TimestampData{
				CreatedAt: timestamp,
			},
		}

		dbError := b.TransactionRepository.SaveTransaction(transaction)
		if dbError != nil {
			slog.Error(dbError.Error())
			utility.InternalServerError(t.context)
			return false
		}

		return true
	}
	return false
}

func (b *BankTransferService) isSuccessfulTransaction(t model.TransactionRequestDTO,
	account *model.Account, ctx *gin.Context) bool {
	switch t.Type {
	case model.DebitTransaction:
		if err := b.handleDebit(t.Amount, account); err != nil {
			slog.Error("debit transaction failed: ", err.Error()) // nolint
			utility.InternalServerError(ctx)
			return false
		}
	case model.CreditTransaction:
		if err := b.handleCredit(t.Amount, account); err != nil {
			slog.Error("credit transaction failed: ", err.Error()) // nolint
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
