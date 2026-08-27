package dao

import (
	"context"
	_ "embed"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

//go:embed seckill.lua
var luaScript string

var Rdb *redis.Client                   // 连 Redis 的管道 (像 DB 之于 MySQL)
var Script = redis.NewScript(luaScript) // 点菜卡, 只建一次

func InitRedis() {
	Rdb = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6380", // 你改过端口, 别写成默认 6379
	})
	if err := Rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Redis connect failed: %v", err)
	}
}

// 秒杀: 拼 key + 跑脚本, 返回 0/1/2
func SecKill(ctx context.Context, uid, pid uint) (int, error) {
	stockKey := fmt.Sprintf("products:stock:%d", pid)
	orderKey := fmt.Sprintf("products:order:%d", pid)
	res, err := Script.Run(ctx, Rdb, []string{stockKey, orderKey}, uid).Int()
	return res, err
}
