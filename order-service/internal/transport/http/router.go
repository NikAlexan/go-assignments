package http

import "github.com/gin-gonic/gin"

func SetupRouter(handler *Handler) *gin.Engine {
	router := gin.Default()
	router.POST("/orders", handler.CreateOrder)
	router.GET("/orders", handler.GetOrdersByStatus)
	router.GET("/orders/:id", handler.GetOrder)
	router.PATCH("/orders/:id/cancel", handler.CancelOrder)
	router.GET("/payments/stats", handler.GetPaymentStats)
	return router
}
