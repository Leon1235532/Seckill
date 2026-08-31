package dao

import (
	"github.com/Leon1235532/Seckill/models"
	"github.com/Leon1235532/Seckill/schemas"
	"gorm.io/gorm"
)

func CreatePdtInfo(pdtinfo *schemas.PdtCreate) (uint, error) {
	product := &models.Product{
		Name:      pdtinfo.Name,
		Stock:     pdtinfo.Stock,
		StartTime: pdtinfo.StartTime,
		EndTime:   pdtinfo.EndTime,
	}
	err := DB.Create(product).Error
	return product.ID, err // GORM 建完后自动把自增主键回填进 product.ID
}

func UpdatePdtInfo(pid uint, info *schemas.PdtUpdate) error {
	res := DB.Model(&models.Product{}).
		Where("id = ?", pid).Updates(info)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func Deletepdt(pid uint) error {
	res := DB.Delete(&models.Product{}, pid)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func QueryPinfo(pid uint) (models.Product, error) {
	var p models.Product
	err := DB.First(&p, pid).Error
	return p, err
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
