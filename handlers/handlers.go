package handlers

import (
	"github.com/Leon1235532/Seckill/dao"
	"github.com/Leon1235532/Seckill/schemas"
	"github.com/Leon1235532/Seckill/service"
	"github.com/gin-gonic/gin"
)

// var mu sync.Mutex

func CreatePdtHandler(c *gin.Context) {
	var pdtinfo schemas.PdtCreate
	if err := c.ShouldBindJSON(&pdtinfo); err != nil {
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Parameter reception failed.",
			"Err":  err,
		})
		c.Abort()
		return
	}
	if err := dao.CreatePdtInfo(&pdtinfo); err != nil {
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Interserver failed!",
			"Err":  err,
		})
		c.Abort()
		return
	}
}

func SaleHandler(c *gin.Context) {
	var info schemas.ProductInfo
	if err := c.ShouldBindJSON(&info); err != nil {
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Parameter reception failed.",
			"Err":  err,
		})
		c.Abort()
		return
	}
	res, err := dao.SecKill(c.Request.Context(), info.Uid, info.Pid)
	if err != nil {
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "interserver failed!",
			"Err":  err,
		})
		c.Abort()
		return
	}
	switch res {
	case 0:
		service.OrderChan <- info
		c.JSON(200, gin.H{
			"code": 200,
			"Msg":  "Congratulations on buying successfully!",
			"Err":  nil,
		})
	case 1:
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Sorry, out of stock!",
			"Err":  nil,
		})
	case 2:
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Each user may purchase this only once!",
			"Err":  nil,
		})
	}

}

//返回 2 代表重复下单，直接拦截
//返回 1 代表库存不足，已经被抢光了
//0成功
