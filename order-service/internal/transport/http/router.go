package http

import (
	"order-service/internal/transport/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func SetupRouter(handler *Handler, rdb *redis.Client, rateLimitRPM int) *gin.Engine {
	router := gin.Default()
	router.Use(middleware.RateLimiter(rdb, rateLimitRPM))
	router.POST("/orders", handler.CreateOrder)
	router.GET("/orders", handler.GetOrdersByStatus)
	router.GET("/orders/:id", handler.GetOrder)
	router.PATCH("/orders/:id/cancel", handler.CancelOrder)
	router.GET("/payments/stats", handler.GetPaymentStats)
	return router
}
