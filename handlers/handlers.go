package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"

	"github.com/Leon1235532/Seckill/cache"
	"github.com/Leon1235532/Seckill/dao"
	"github.com/Leon1235532/Seckill/rabbitmq"
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
	id, err := dao.CreatePdtInfo(&pdtinfo)
	if err != nil {
		FailResponse(c, 500, ServerMsg, err)
		return
	}
	// info同步保存到 Redis缓存
	if err := cache.AddInfoCache(id); err != nil {
		FailResponse(c, 500, "Preload activity failed!", err)
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
	err = dao.UpdatePdtInfo(uint(pid), &modifyinfo)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		FailResponse(c, 404, FindMsg, err)
		return
	}
	if err != nil {
		FailResponse(c, 500, ServerMsg, err)
		return
	}

	if err := cache.ModifyCache(uint(pid), &modifyinfo); err != nil {
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
	err = dao.Deletepdt(uint(pid))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		FailResponse(c, 404, FindMsg, err)
		return
	}
	if err != nil {
		FailResponse(c, 500, ServerMsg, err)
		return
	}
	if err := cache.DeleteCache(uint(pid)); err != nil {
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

	// 熔断器状态为Open时，直接拦截，不进redis，不扣库存
	if rabbitmq.MQOpen() {
		FailResponse(c, 503, "System busy, please try again later!", nil)
		return
	}

	res, err := cache.SecKill(c.Request.Context(), info.Uid, info.Pid)
	if err != nil {
		FailResponse(c, 500, ServerMsg, err)
		return
	}
	// 0:成功，1:库存不足，2:重复下单，3:时间未开始，4:活动已结束
	switch res {
	case 0:
		body, err := json.Marshal(info)
		if err != nil {
			log.Printf("struct marshal failed:%v", err)
		}

		if err := rabbitmq.PublishWithBreaker("seckill_queue", body, 3); err != nil {
			if compErr := cache.CompensateSecKill(info.Uid, info.Pid); compErr != nil {
				log.Printf("Order Compensate Failed, uid=%d pid=%d: %v",
					info.Uid, info.Pid, compErr)
			}
			// 三次重试资源耗尽才报错，连续报错五次后触发熔断，状态转为Open
			FailResponse(c, 500, "MQ abnormal, please check it!", err)
			return
		}
		Success(c, "Congratulations on buying successfully!", nil)

	case 1:
		FailResponse(c, 400, "Sorry, out of stock!", nil)

	case 2:
		FailResponse(c, 400, "Each user may purchase this only once!", nil)

	case 3:
		FailResponse(c, 400, "Flash sale has not yet started!", nil)

	case 4:
		FailResponse(c, 400, "The event has ended!", nil)
	}
}
