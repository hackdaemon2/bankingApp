package handler

import "github.com/gin-gonic/gin"

type IBankTransferService interface {
	StatusQuery(context *gin.Context)
	Transfer(context *gin.Context)
}

func NewBankTransferHandler(service IBankTransferService) *BankTransferHandler {
	return &BankTransferHandler{
		BankTransferService: service,
	}
}

type BankTransferHandler struct {
	BankTransferService IBankTransferService
}

func (b *BankTransferHandler) Transfer(context *gin.Context) {
	b.BankTransferService.Transfer(context)
}

func (b *BankTransferHandler) StatusQuery(context *gin.Context) {
	b.BankTransferService.StatusQuery(context)
}
