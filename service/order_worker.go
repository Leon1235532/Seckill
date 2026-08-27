package service

import (
	"log"

	"github.com/Leon1235532/Seckill/dao"
	"github.com/Leon1235532/Seckill/schemas"
)

var OrderChan = make(chan schemas.ProductInfo, 1024)

func StartWorker(n int) {
	for i := 0; i < n; i++ {
		go func() {
			for info := range OrderChan {
				if err := dao.InsertOrder(info.Uid, info.Pid); err != nil {
					log.Printf("insert order failed uid=%d pid=%d: %v", info.Uid, info.Pid, err)
				}
			}
		}()
	}
}
