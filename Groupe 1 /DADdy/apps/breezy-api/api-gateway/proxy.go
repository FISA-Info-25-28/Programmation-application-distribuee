package main

import (
	"log"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
)

func proxyHandler(proxy *httputil.ReverseProxy) gin.HandlerFunc {
	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func requestLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("%s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
	}
}
