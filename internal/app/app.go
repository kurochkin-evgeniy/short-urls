package app

import (
	"net/http"
	"short-urls/internal/handler"
	"short-urls/internal/service"

	"github.com/gin-gonic/gin"
)

type ShortUrlApp struct {
	shortUrlService *service.ShortUrlService
}

func NewShortUrlApp() *ShortUrlApp {
	return &ShortUrlApp{shortUrlService: service.NewShortUrlService()}
}

func (a *ShortUrlApp) Start() error {

	r := gin.Default()

	r.POST("/", gin.WrapH(http.HandlerFunc(handler.HandleRequest(a.shortUrlService))))
	r.GET("/:id", gin.WrapH(http.HandlerFunc(handler.HandleRequest(a.shortUrlService))))

	return r.Run(":8080")
}
