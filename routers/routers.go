package routers

import (
	"github.com/Leon1235532/Seckill/handlers"
	"github.com/Leon1235532/Seckill/setting"
	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	if setting.Conf.Release {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.POST("/addpdt", handlers.CreatePdtHandler)
	r.POST("/seckill", handlers.SaleHandler)
	return r
}
