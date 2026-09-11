package service

import (
	"encoding/json"
	"log"

	"github.com/Leon1235532/Seckill/dao"
	"github.com/Leon1235532/Seckill/rabbitmq"
	"github.com/Leon1235532/Seckill/schemas"
)

func StartWorker(n int) {
	for i := 0; i < n; i++ {
		go func() {
			rmq := rabbitmq.NewRabbitMQWork("seckill_queue")
			defer rmq.Destory()
			rmq.ReceiveWork(func(msg []byte) error {
				var info schemas.PrdOrderInfo
				if err := json.Unmarshal(msg, &info); err != nil {
					log.Printf("数据无法解析:%v", err)
					return err
				}
				if err := dao.InsertOrder(info.Uid, info.Pid); err != nil {
					log.Printf("insert order failed uid=%d pid=%d: %v", info.Uid, info.Pid, err)
					return err
				}
				return nil
			})
		}()
	}
}
