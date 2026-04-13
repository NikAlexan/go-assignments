package http

import "github.com/gin-gonic/gin"

func SetupRouter(handler *Handler) *gin.Engine {
	router := gin.Default()
	router.POST("/payments", handler.Authorize)
	router.GET("/payments/:order_id", handler.GetByOrderID)
	return router
}
