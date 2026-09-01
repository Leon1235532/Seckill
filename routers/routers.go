package routers

import (
	"github.com/Leon1235532/Seckill/bocket"
	"github.com/Leon1235532/Seckill/handlers"
	"github.com/Leon1235532/Seckill/setting"
	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	if setting.Conf.Release {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.GET("/checkpdt/:pid", handlers.CheckHandler)
	r.POST("/addpdt", handlers.CreatePdtHandler)
	r.POST("/seckill", bocket.RateLimiter(bocket.NewTokenBucket(100, 100)), handlers.SaleHandler)
	r.POST("/modify/:pid", handlers.UpdatePdtHandler)
	r.DELETE("/delete/:pid", handlers.DeleteHandler)
	return r
}
