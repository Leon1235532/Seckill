package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/Leon1235532/Seckill/cache"
	"github.com/Leon1235532/Seckill/rabbitmq"
	"github.com/Leon1235532/Seckill/schemas"
)

var (
	ErrMQOpen   = errors.New("mq breaker open")
	ErrSoldOut  = errors.New("sold out")
	ErrDupOrder = errors.New("duplicate order")
	ErrNotStart = errors.New("not started")
	ErrEnded    = errors.New("ended")
)

func Seckill(ctx context.Context, uid, pid uint) error {
	// 熔断器状态为Open时，直接拦截，不进redis，不扣库存
	if rabbitmq.MQOpen() {
		return ErrMQOpen
	}
	res, err := cache.SecKill(ctx, uid, pid)
	if err != nil {
		return err
	}
	// 0:成功，1:库存不足，2:重复下单，3:时间未开始，4:活动已结束
	switch res {
	case 0:
		body, err := json.Marshal(schemas.PrdOrderInfo{Uid: uid, Pid: pid})
		if err != nil {
			return err
		}
		// 三次重试资源耗尽才报错，连续报错五次后触发熔断，状态转为Open
		if err := rabbitmq.PublishWithBreaker("seckill_queue", body, 3); err != nil {
			if compErr := cache.CompensateSecKill(uid, pid); compErr != nil {
				log.Printf("Order Compensate Failed, uid=%d pid=%d: %v", uid, pid, compErr)
			}
			return fmt.Errorf("MQ abnormal! failed: %w", err)
		}
		return nil
	case 1:
		return ErrSoldOut
	case 2:
		return ErrDupOrder
	case 3:
		return ErrNotStart
	case 4:
		return ErrEnded
	}
	return fmt.Errorf("unknown seckill result: %d", res)
}
