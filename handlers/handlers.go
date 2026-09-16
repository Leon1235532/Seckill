package handlers

import (
	"errors"
	"strconv"

	"github.com/Leon1235532/Seckill/schemas"
	"github.com/Leon1235532/Seckill/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreatePdtHandler(c *gin.Context) {
	var pdtinfo schemas.PdtCreate
	if err := c.ShouldBindJSON(&pdtinfo); err != nil {
		FailResponse(c, 400, ParaMsg, err)
		return
	}
	id, err := service.CreatePdt(&pdtinfo)
	if err != nil {
		FailResponse(c, 500, ServerMsg, err)
		return
	}

	Success(c, "Product created!", id)
}

func UpdatePdtHandler(c *gin.Context) {
	var modifyinfo schemas.PdtUpdate
	pidstr := c.Param("pid")
	pid, err := strconv.Atoi(pidstr)
	if err != nil || pid < 0 {
		FailResponse(c, 400, ParaPidMsg, err)
		return
	}
	if err := c.ShouldBindJSON(&modifyinfo); err != nil {
		FailResponse(c, 400, ParaMsg, err)
		return
	}

	err = service.UpdatePdt(uint(pid), &modifyinfo)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		FailResponse(c, 404, FindMsg, err)
		return
	}

	if err != nil {
		FailResponse(c, 500, ServerMsg, err)
		return
	}

	Success(c, "Product info modified!", pid)
}

func DeleteHandler(c *gin.Context) {
	pidstr := c.Param("pid")
	pid, err := strconv.Atoi(pidstr)
	if err != nil || pid < 0 {
		FailResponse(c, 400, ParaPidMsg, err)
		return
	}
	err = service.DeletePdt(uint(pid))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		FailResponse(c, 404, FindMsg, err)
		return
	}
	if err != nil {
		FailResponse(c, 500, ServerMsg, err)
		return
	}

	Success(c, "Product delete succeed!", pid)
}

// 查询商品信息接口，缓存不存在触发单飞模式拿数据库
func CheckHandler(c *gin.Context) {
	pidstr := c.Param("pid")
	pid, err := strconv.Atoi(pidstr)
	if err != nil || pid < 0 {
		FailResponse(c, 400, ParaPidMsg, err)
		return
	}
	data, err := service.GetPdtInfoByOne(uint(pid))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		FailResponse(c, 404, FindMsg, err)
		return
	}
	if err != nil {
		FailResponse(c, 500, ServerMsg, err)
		return
	}
	c.Data(200, "application/json", data)
}

func SaleHandler(c *gin.Context) {
	var info schemas.PrdOrderInfo
	if err := c.ShouldBindJSON(&info); err != nil {
		FailResponse(c, 400, ParaMsg, err)
		return
	}

	err := service.Seckill(c.Request.Context(), info.Uid, info.Pid)
	switch {
	case errors.Is(err, service.ErrMQOpen):
		FailResponse(c, 503, "System busy, please try again later!", nil)
	case errors.Is(err, service.ErrSoldOut):
		FailResponse(c, 400, "Sorry, out of stock!", nil)
	case errors.Is(err, service.ErrDupOrder):
		FailResponse(c, 400, "Each user may purchase this only once!", nil)
	case errors.Is(err, service.ErrNotStart):
		FailResponse(c, 400, "Flash sale has not yet started!", nil)
	case errors.Is(err, service.ErrEnded):
		FailResponse(c, 400, "The event has ended!", nil)
	case err != nil:
		FailResponse(c, 500, ServerMsg, err)
	default:
		Success(c, "Congratulations on buying successfully!", nil)
	}
}
