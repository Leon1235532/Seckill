package schemas

type ProductInfo struct {
	Pid uint `binding:"required" json:"pid"`
	Uid uint `binding:"required" json:"uid"`
}

type PdtCreate struct {
	Name  string `binding:"required" json:"name"`
	Stock int    `json:"stock"`
}
