package models

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"not null" json:"name"`
	Stock     int            `gorm:"not null" json:"stock"` // 真实库存
	StartTime *time.Time     `json:"starttime"`
	EndTime   *time.Time     `json:"endtime"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Order struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_user_product" json:"user_id"`          // 压测脚本随机生成的虚拟用户ID
	ProductID uint      `gorm:"index;uniqueIndex:uk_user_product" json:"product_id"` // 关联商品
	Status    int       `gorm:"type:tinyint;default:1" json:"status"`                // 1: 创建成功
	CreatedAt time.Time `json:"created_at"`
}
