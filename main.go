package main

import (
	"fmt"
	"log"

	"github.com/Leon1235532/Seckill/dao"
	"github.com/Leon1235532/Seckill/models"
	"github.com/Leon1235532/Seckill/routers"
	"github.com/Leon1235532/Seckill/service"
	"github.com/Leon1235532/Seckill/setting"
)

const FilePath = "./config/config.ini"

func main() {
	if err := setting.Init(FilePath); err != nil {
		log.Fatalf("load mysql config failed: %#v", err.Error())
	}
	if err := dao.InitDB(setting.Conf.MySQLConfig); err != nil {
		log.Fatalf("MySQL init failed: %#v", err.Error())
	}
	if err := dao.DB.AutoMigrate(&models.Product{}, &models.Order{}); err != nil {
		log.Fatalf("create tables failed: %#v", err.Error())
	}
	dao.InitRedis()
	service.StartWorker(10)
	r := routers.Router()
	if err := r.Run(fmt.Sprintf(":%d", setting.Conf.Port)); err != nil {
		log.Fatalf("router register failed: %#v", err.Error())
	}
	defer dao.Close()
}
