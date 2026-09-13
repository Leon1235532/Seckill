package service

import (
	"encoding/json"
	"log"
	"time"

	"github.com/Leon1235532/Seckill/dao"
	"github.com/Leon1235532/Seckill/rabbitmq"
	"github.com/Leon1235532/Seckill/schemas"
)

func StartWorker(n int) {
	for i := 1; i <= n; i++ {
		go func(num int) {
			//状态标志
			isReconnecting := false
			for {
				rmq, err := rabbitmq.NewRabbitMQWork("seckill_queue")
				if err != nil {
					if !isReconnecting {
						log.Printf("consumer-%d 连接失败:%v, 3s后重试", num, err)
						isReconnecting = true
					}
					time.Sleep(3 * time.Second)
					continue
				}

				if isReconnecting {
					log.Printf("✅ consumer-%d 🔄 重连成功，恢复监听!", num)
					isReconnecting = false // 重置状态
				} else {
					// 首次启动成功
					log.Printf("consumer-%d 已连接,开始监听队列", num)
				}

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

				//连接正常时ReceiveWork会持续监听，退出则说明连接异常了
				rmq.Destory()
				log.Printf("consumer-%d 连接断开,3s后重连", num)
				time.Sleep(3 * time.Second)
			}
		}(i)
	}
	log.Printf("%d 个RabbitMQ 消费者后台协程初始化完毕,正在监听队列！", n)
}
