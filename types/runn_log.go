package types

type GetRunLogRequest struct {
	Trx string `json:"trx" form:"trx" binding:"required"`
}
