package handler

import "github.com/gin-gonic/gin"

type IBankTransferService interface {
	StatusQuery(context *gin.Context)
	Transfer(context *gin.Context)
}

type BankTransferHandler struct {
	BankTransferService IBankTransferService
}

func NewBankTransferHandler(service IBankTransferService) *BankTransferHandler {
	return &BankTransferHandler{
		BankTransferService: service,
	}
}

func (b *BankTransferHandler) Transfer(context *gin.Context) {
	b.BankTransferService.Transfer(context)
}

func (b *BankTransferHandler) StatusQuery(context *gin.Context) {
	b.BankTransferService.StatusQuery(context)
}
