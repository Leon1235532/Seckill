package handlers

import (
	"errors"
	"strconv"

	"github.com/Leon1235532/Seckill/dao"
	"github.com/Leon1235532/Seckill/schemas"
	"github.com/Leon1235532/Seckill/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreatePdtHandler(c *gin.Context) {
	var pdtinfo schemas.PdtCreate
	if err := c.ShouldBindJSON(&pdtinfo); err != nil {
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Parameter reception failed.",
			"Err":  err,
		})
		return
	}
	id, err := dao.CreatePdtInfo(&pdtinfo)
	if err != nil {
		c.JSON(500, gin.H{
			"code": 500,
			"Msg":  "Interserver failed!",
			"Err":  err,
		})
		return
	}
	// 编排逻辑: 建完户口本, 复印一份到 Redis (预热只发生在这里, 不在抢购路径上)
	if err := dao.PreloadActivity(id); err != nil {
		c.JSON(500, gin.H{
			"code": 500,
			"Msg":  "Preload activity failed!",
			"Err":  err,
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"Msg":  "Product created.",
		"ID":   id,
	})
}

func UpdatePdtHandler(c *gin.Context) {
	var modifyinfo schemas.PdtUpdate
	pidstr := c.Param("pid")
	pid, err := strconv.Atoi(pidstr)
	if err != nil || pid < 0 {
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Parameter reception failed or pid is not uint",
			"Err":  err,
		})
		return
	}
	if err := c.ShouldBindJSON(&modifyinfo); err != nil {
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Parameter reception failed.",
			"Err":  err,
		})
		return
	}
	err = dao.UpdatePdtInfo(uint(pid), &modifyinfo)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(404, gin.H{
			"code": 404,
			"Msg":  "Product not found.",
		})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{
			"code": 500,
			"Msg":  "Interserver failed!",
			"Err":  err,
		})
		return
	}

	if err := dao.ModifyRedis(uint(pid), &modifyinfo); err != nil {
		c.JSON(500, gin.H{
			"code": 500,
			"Msg":  "Interserver failed!",
			"Err":  err,
		})
		return
	}

	c.JSON(200, gin.H{
		"code": 200,
		"Msg":  "Product info modified.",
		"ID":   pid,
	})
}

func DeleteHandler(c *gin.Context) {
	pidstr := c.Param("pid")
	pid, err := strconv.Atoi(pidstr)
	if err != nil || pid < 0 {
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Parameter reception failed or pid is not uint",
			"Err":  err,
		})
		return
	}
	err = dao.Deletepdt(uint(pid))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(404, gin.H{
			"code": 404,
			"Msg":  "Product not found!",
			"Err":  err,
		})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{
			"code": 500,
			"Msg":  "Interserver failed!",
			"Err":  err,
		})
		return
	}
	if err := dao.DeleteRedis(uint(pid)); err != nil {
		c.JSON(500, gin.H{
			"code": 500,
			"Msg":  "Interserver failed!",
			"Err":  err,
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"Msg":  "Product delete succeed!",
		"ID":   pid,
	})
}

func CheckHandler(c *gin.Context) {
	pidstr := c.Param("pid")
	pid, err := strconv.Atoi(pidstr)
	if err != nil || pid < 0 {
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Parameter reception failed or pid is not uint",
			"Err":  err,
		})
		return
	}
	data, err := service.GetPdtInfo(uint(pid))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(404, gin.H{
			"code": 404,
			"Msg":  "Product not found!",
			"err":  err,
		})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{
			"code": 500,
			"Msg":  "Interserver failed!",
			"Err":  err,
		})
		return
	}
	c.Data(200, "application/json", data)
}

func SaleHandler(c *gin.Context) {
	var info schemas.PrdOrderInfo
	if err := c.ShouldBindJSON(&info); err != nil {
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Parameter reception failed.",
			"Err":  err,
		})
		return
	}
	res, err := dao.SecKill(c.Request.Context(), info.Uid, info.Pid)
	if err != nil {
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "interserver failed!",
			"Err":  err,
		})
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
	case 3:
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "Flash sale has not yet started!",
			"Err":  nil,
		})
	case 4:
		c.JSON(400, gin.H{
			"code": 400,
			"Msg":  "The event has ended!",
			"Err":  nil,
		})
	}

}

//返回 2 代表重复下单，直接拦截
//返回 1 代表库存不足，已经被抢光了
//0成功，1代表库存不足，2代表重复下单，3代表时间未开始。4代表活动已结束
