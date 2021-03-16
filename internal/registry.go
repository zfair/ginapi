package internal

import "github.com/gin-gonic/gin"

type GinRegistry struct {
	HttpMethod  string
	URL         string
	Main        gin.HandlerFunc
	Middlewares []gin.HandlerFunc
}
