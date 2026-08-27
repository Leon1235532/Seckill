package dao

import (
	"fmt"
	"log"

	"github.com/Leon1235532/Seckill/setting"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(cfg *setting.MySQLConfig) (err error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DB)
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("MySQL connect failed: %v", err)
	}
	return
}

func Close() (err error) {
	sqldb, err := DB.DB()
	if err != nil {
		panic(err)
	}
	sqldb.Close()
	return
}
