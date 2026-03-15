package app

import (
	"net/http"
	"short-urls/internal/config"
	"short-urls/internal/handler"
	"short-urls/internal/service"

	"github.com/gin-gonic/gin"
)

type ShortUrlApp struct {
	cfg             *config.Config
	shortUrlService *service.ShortUrlService
}

func NewShortUrlApp() *ShortUrlApp {
	cfg := config.NewConfig()
	return &ShortUrlApp{
		shortUrlService: service.NewShortUrlService(cfg.BaseUrl),
		cfg:             cfg,
	}
}

func (a *ShortUrlApp) Start() error {

	r := gin.Default()

	r.POST("/", gin.WrapH(http.HandlerFunc(handler.HandleRequest(a.shortUrlService))))
	r.GET("/:id", gin.WrapH(http.HandlerFunc(handler.HandleRequest(a.shortUrlService))))

	return r.Run(a.cfg.HostAddr)
}
