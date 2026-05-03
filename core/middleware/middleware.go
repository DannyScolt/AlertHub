package middleware

import "github.com/gin-gonic/gin"

func RegisterGlobal(router *gin.Engine) {
	router.Use(RequestID())
	router.Use(gin.Recovery())
}
