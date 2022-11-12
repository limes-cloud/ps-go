package types

type GetScriptRequest struct {
	ID      int64  `json:"id" form:"id"`
	Name    string `json:"name" form:"name"`
	Version string `json:"version" form:"version"`
}

type PageScriptRequest struct {
	Page  int `json:"page" form:"page" binding:"required" sql:"-"`
	Count int `json:"count" form:"count"  binding:"required,max=50"  sql:"-"`

	Status     *bool  `json:"status" form:"status"`
	Name       string `json:"name" form:"name"`
	OperatorID int64  `json:"operator_id" form:"operator_id"`
	Start      int64  `json:"start" form:"start" sql:"> ?" field:"created_at"`
	End        int64  `json:"end" form:"end" sql:"< ?" field:"created_at"`
}

type AddScriptRequest struct {
	Name       string `json:"name" binding:"required"`
	Script     string `json:"script" binding:"required"`
	Operator   string `json:"operator" binding:"required"`
	OperatorID int64  `json:"operator_id" binding:"required"`
}

type SwitchVersionScriptRequest struct {
	ID         int64  `json:"id" binding:"required"`
	Operator   string `json:"operator"`
	OperatorID int64  `json:"operator_id"`
}

type DeleteScriptRequest struct {
	ID         int64  `json:"id" binding:"required"`
	Operator   string `json:"operator" binding:"required"`
	OperatorID int64  `json:"operator_id" binding:"required"`
}
