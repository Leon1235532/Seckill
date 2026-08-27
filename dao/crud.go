package dao

import (
	"github.com/Leon1235532/Seckill/models"
	"github.com/Leon1235532/Seckill/schemas"
	"gorm.io/gorm"
)

func CreatePdtInfo(pdtinfo *schemas.PdtCreate) error {
	product := &models.Product{
		Name:  pdtinfo.Name,
		Stock: pdtinfo.Stock,
	}
	err := DB.Create(product).Error
	return err
}

func QueryStock(pid uint) (int, error) {
	var stock int
	err := DB.Model(&models.Product{}).Where("id = ?", pid).
		Select("stock").Scan(&stock).Error
	return stock, err
}

func InsertOrder(uid, pid uint) error {
	order := &models.Order{
		UserID:    uid,
		ProductID: pid,
		Status:    1,
	}
	if err := DB.Create(order).Error; err != nil {
		return err
	}
	err := DB.Model(&models.Product{}).Where("id = ?", pid).
		Update("stock", gorm.Expr("stock - ?", 1)).Error
	return err
}
