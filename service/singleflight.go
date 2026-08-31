package service

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/Leon1235532/Seckill/dao"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

var g singleflight.Group

func GetPdtInfo(pid uint) ([]byte, error) {
	val, err := dao.GetRedis(pid)
	if err == nil {
		return val, nil
	}
	if !errors.Is(err, redis.Nil) {
		return nil, err
	}

	fn := func() (any, error) {
		p, err := dao.QueryPinfo(pid) // 裸IO
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(p)
		if err != nil {
			return nil, err
		}
		if err := dao.SetInfoCache(pid, data); err != nil { // 回填失败别吞, 但也别因它不返回数据
			return nil, err
		}
		return data, nil
	}

	v, err, _ := g.Do(strconv.Itoa(int(pid)), fn)
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}
