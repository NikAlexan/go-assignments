package http

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"order-service/internal/domain"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type orderUseCase interface {
	CreateOrder(ctx context.Context, input usecase.CreateOrderInput) (*domain.Order, error)
	GetOrder(ctx context.Context, id string) (*domain.Order, error)
	CancelOrder(ctx context.Context, id string) (*domain.Order, error)
	GetOrdersByStatus(ctx context.Context, status string) ([]*domain.Order, error)
}

type Handler struct {
	useCase orderUseCase
}

func NewHandler(useCase orderUseCase) *Handler {
	return &Handler{useCase: useCase}
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name"   binding:"required"`
	Amount     int64  `json:"amount"      binding:"required,gt=0"`
}

func orderResponse(order *domain.Order) gin.H {
	return gin.H{
		"id":          order.ID,
		"customer_id": order.CustomerID,
		"item_name":   order.ItemName,
		"amount":      order.Amount,
		"status":      order.Status,
		"created_at":  order.CreatedAt,
	}
}

func (handler *Handler) CreateOrder(c *gin.Context) {
	var request createOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input := usecase.CreateOrderInput{
		CustomerID:     request.CustomerID,
		ItemName:       request.ItemName,
		Amount:         request.Amount,
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
	}

	order, err := handler.useCase.CreateOrder(c.Request.Context(), input)
	if errors.Is(err, usecase.ErrPaymentUnavailable) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "payment service unavailable",
			"order": orderResponse(order),
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, orderResponse(order))
}

func (handler *Handler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	order, err := handler.useCase.GetOrder(c.Request.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orderResponse(order))
}

func (handler *Handler) CancelOrder(c *gin.Context) {
	id := c.Param("id")
	order, err := handler.useCase.CancelOrder(c.Request.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if errors.Is(err, domain.ErrCannotCancel) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orderResponse(order))
}

var validStatuses = map[string]bool{
	"Pending":   true,
	"Paid":      true,
	"Failed":    true,
	"Cancelled": true,
}

func (handler *Handler) GetOrdersByStatus(c *gin.Context) {
	status := c.Query("status")
	if status == "" || !validStatuses[status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status type"})
		return
	}

	orders, err := handler.useCase.GetOrdersByStatus(c.Request.Context(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]gin.H, len(orders))
	for i, o := range orders {
		result[i] = orderResponse(o)
	}
	c.JSON(http.StatusOK, result)
}
