package service

import (
	"fmt"

	"github.com/Leon1235532/Seckill/cache"
	"github.com/Leon1235532/Seckill/dao"
	"github.com/Leon1235532/Seckill/schemas"
)

func CreatePdt(pdtinfo *schemas.PdtCreate) (uint, error) {
	id, err := dao.CreatePdtInfo(pdtinfo)
	if err != nil {
		return 0, err
	}
	// info同步保存到 Redis缓存
	if err := cache.AddInfoCache(id); err != nil {
		return id, fmt.Errorf("db created but cache sync failed: %w", err)
	}
	return id, nil
}

func UpdatePdt(pid uint, info *schemas.PdtUpdate) error {
	if err := dao.UpdatePdtInfo(pid, info); err != nil {
		return err
	}
	// 同步改缓存，删info
	if err := cache.ModifyCache(pid, info); err != nil {
		return fmt.Errorf("db updated but cache sync failed: %w", err)
	}
	return nil
}

func DeletePdt(pid uint) error {
	if err := dao.Deletepdt(uint(pid)); err != nil {
		return err
	}
	if err := cache.DeleteCache(uint(pid)); err != nil {
		return fmt.Errorf("db deleted but cache sync failed: %w", err)
	}
	return nil
}
