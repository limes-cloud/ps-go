package types

type GetRuleRequest struct {
	ID      int64  `json:"id" form:"id"`
	Name    string `json:"name" form:"name"`
	Method  string `json:"method" form:"method"`
	Version string `json:"version" form:"version"`
}

type PageRuleRequest struct {
	Page       int    `json:"page" form:"page" binding:"required" sql:"-"`
	Count      int    `json:"count" form:"count"  binding:"required,max=50"  sql:"-"`
	Method     string `json:"method" form:"method"`
	Status     *bool  `json:"status" form:"status"`
	Name       string `json:"name" form:"name"`
	OperatorID int64  `json:"operator_id" form:"operator_id"`
	Start      int64  `json:"start" form:"start" sql:"> ?" field:"created_at"`
	End        int64  `json:"end" form:"end" sql:"< ?" field:"created_at"`
}

type AddRuleRequest struct {
	Name       string `json:"name" binding:"required"`
	Rule       string `json:"rule" binding:"required"`
	Method     string `json:"method"  binding:"required"`
	Operator   string `json:"operator" binding:"required"`
	OperatorID int64  `json:"operator_id" binding:"required"`
}

type SwitchVersionRuleRequest struct {
	ID         int64  `json:"id" binding:"required"`
	Operator   string `json:"operator"`
	OperatorID int64  `json:"operator_id"`
}

type DeleteRuleRequest struct {
	ID         int64  `json:"id" binding:"required"`
	Operator   string `json:"operator" binding:"required"`
	OperatorID int64  `json:"operator_id" binding:"required"`
}
