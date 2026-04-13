package http

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"payment-service/internal/domain"

	"github.com/gin-gonic/gin"
)

type paymentUseCase interface {
	Authorize(ctx context.Context, orderID string, amount int64) (*domain.Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}

type Handler struct {
	useCase paymentUseCase
}

func NewHandler(useCase paymentUseCase) *Handler {
	return &Handler{useCase: useCase}
}

type authorizeRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Amount  int64  `json:"amount"   binding:"required,gt=0"`
}

func (handler *Handler) Authorize(c *gin.Context) {
	var request authorizeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := handler.useCase.Authorize(c.Request.Context(), request.OrderID, request.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":             payment.ID,
		"order_id":       payment.OrderID,
		"transaction_id": payment.TransactionID,
		"amount":         payment.Amount,
		"status":         payment.Status,
	})
}

func (handler *Handler) GetByOrderID(c *gin.Context) {
	orderID := c.Param("order_id")

	payment, err := handler.useCase.GetByOrderID(c.Request.Context(), orderID)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             payment.ID,
		"order_id":       payment.OrderID,
		"transaction_id": payment.TransactionID,
		"amount":         payment.Amount,
		"status":         payment.Status,
	})
}
