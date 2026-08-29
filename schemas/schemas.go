package schemas

import "time"

type ProductInfo struct {
	Pid uint `binding:"required" json:"pid"`
	Uid uint `binding:"required" json:"uid"`
}

type PdtCreate struct {
	Name      string    `binding:"required" json:"name"`
	Stock     int       `json:"stock"`
	StartTime time.Time `json:"starttime"`
	EndTime   time.Time `json:"endtime"`
}
