package handler

import (
	"bankingApp/internal/api/constants"
	"bankingApp/internal/utility"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NotFoundHandler(c *gin.Context) {
	c.JSON(http.StatusNotFound, utility.FormulateErrorResponse(constants.ResourceNotFoundMsg))
}

func NoMethodHandler(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, utility.FormulateErrorResponse(constants.MethodNotAllowed))
}
