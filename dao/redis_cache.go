package dao

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"time"

	"github.com/Leon1235532/Seckill/schemas"
	"github.com/Leon1235532/Seckill/setting"
	"github.com/redis/go-redis/v9"
)

//go:embed seckill.lua
var luaScript string

var Rdb *redis.Client                   // 连 Redis 的管道 (像 DB之于MySQL )
var Script = redis.NewScript(luaScript) // 点菜卡, 只建一次

func InitRedis(cfg *setting.RedisConfig) {
	Rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	})
	if err := Rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Redis connect failed: %v", err)
	}
}

// 秒杀: 拼 key + 跑脚本, 返回 0/1/2
func SecKill(ctx context.Context, uid, pid uint) (int, error) {
	stockKey := fmt.Sprintf("products:stock:%d", pid)
	orderKey := fmt.Sprintf("products:order:%d", pid)
	startKey := fmt.Sprintf("products:start:%d", pid)
	endKey := fmt.Sprintf("products:end:%d", pid)
	res, err := Script.Run(ctx, Rdb, []string{stockKey, orderKey, startKey, endKey}, uid).Int()
	return res, err
}

func ModifyRedis(pid uint, info *schemas.PdtUpdate) error {
	ctx := context.Background()
	if err := Rdb.Del(ctx, fmt.Sprintf("products:info:%d", pid)).Err(); err != nil {
		log.Printf("del info cache pid=%d: %v", pid, err)
	}
	if info.Stock != nil {
		if err := Rdb.Set(ctx, fmt.Sprintf("products:stock:%d", pid),
			*info.Stock, 0).Err(); err != nil {
			return err
		}
	}

	if info.StartTime != nil {
		if err := Rdb.Set(ctx,
			fmt.Sprintf("products:start:%d", pid),
			info.StartTime.UnixMilli(), 0).Err(); err != nil {
			return err
		}
	}

	if info.EndTime != nil {
		return Rdb.Set(ctx,
			fmt.Sprintf("products:end:%d", pid),
			info.EndTime.UnixMilli(), 0).Err()
	}
	return nil // 什么都没改 → 缓存本来就没错, 什么都不用做
}

func DeleteRedis(pid uint) error {
	ctx := context.Background()
	if err := Rdb.Del(ctx, fmt.Sprintf("products:info:%d", pid)).Err(); err != nil {
		log.Printf("del info cache pid=%d: %v", pid, err)
	}
	if err := Rdb.Del(ctx, fmt.Sprintf("products:stock:%d", pid)).Err(); err != nil {
		return err
	}
	if err := Rdb.Del(ctx, fmt.Sprintf("products:start:%d", pid)).Err(); err != nil {
		return err
	}
	if err := Rdb.Del(ctx, fmt.Sprintf("products:end:%d", pid)).Err(); err != nil {
		return err
	}
	return nil
}

func GetRedis(pid uint) ([]byte, error) {
	val, err := Rdb.Get(context.Background(),
		fmt.Sprintf("products:info:%d", pid)).Bytes()
	if err != nil {
		return nil, err
	}
	return val, nil
}

func SetInfoCache(pid uint, data []byte) error { // 裸SETEX
	ctx := context.Background()
	return Rdb.Set(ctx, fmt.Sprintf("products:info:%d", pid), data, time.Minute).Err()
}

func PreloadActivity(pid uint) error { // 1. 备一个空的结构体准备装货
	p, err := QueryPinfo(pid)
	if err != nil {
		return err
	}

	if err := Rdb.Set(context.Background(),
		fmt.Sprintf("products:stock:%d", p.ID),
		p.Stock, 0).Err(); err != nil {
		return err
	}

	if p.StartTime != nil {
		if err := Rdb.Set(context.Background(), // 把开始时间换成毫秒塞进 Redis
			fmt.Sprintf("products:start:%d", pid),
			p.StartTime.UnixMilli(), //    值 = "1787665680542" 这样的字符串
			0).Err(); err != nil {   //    ⇨ 原生命令: SET products:start:1 1787665680542
			return err
		}
	}

	if p.EndTime != nil {
		return Rdb.Set(context.Background(),
			fmt.Sprintf("products:end:%d", pid),
			p.EndTime.UnixMilli(),
			0).Err() //    ⇨ SET products:end:1 ...
	}
	return nil
}
