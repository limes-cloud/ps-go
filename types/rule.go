package types

type GetRuleRequest struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type PageRuleRequest struct {
	Page  int `json:"page" form:"page" binding:"required" sql:"-"`
	Count int `json:"count" form:"count"  binding:"required,max=50"  sql:"-"`

	Name       string `json:"name" form:"name"`
	OperatorID int64  `json:"operator_id" form:"operator_id"`
	Start      int64  `json:"start" form:"start" sql:"> ?" field:"created_at"`
	End        int64  `json:"end" form:"end" sql:"< ?" field:"created_at"`
}

type AddRuleRequest struct {
	Name       string `json:"name" binding:"required"`
	Rule       string `json:"rule" binding:"required"`
	Operator   string `json:"operator" binding:"required"`
	OperatorID int64  `json:"operator_id" binding:"required"`
}

type UpdateRuleRequest struct {
	ID         int64  `json:"id" binding:"required"`
	Name       string `json:"name"`
	Rule       string `json:"rule"`
	Operator   string `json:"operator"`
	OperatorID int64  `json:"operator_id"`
}

type DeleteRuleRequest struct {
	Name       string `json:"name" binding:"required"`
	Operator   string `json:"operator" binding:"required"`
	OperatorID int64  `json:"operator_id" binding:"required"`
}
