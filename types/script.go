package types

type GetScriptRequest struct {
	ID   int64  `json:"id" form:"id"`
	Name string `json:"name" form:"name"`
}

type PageScriptRequest struct {
	Page  int `json:"page" form:"page" binding:"required" sql:"-"`
	Count int `json:"count" form:"count"  binding:"required,max=50"  sql:"-"`

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

type UpdateScriptRequest struct {
	ID         int64  `json:"id" binding:"required"`
	Name       string `json:"name"`
	Script     string `json:"script"`
	Operator   string `json:"operator"`
	OperatorID int64  `json:"operator_id"`
}

type DeleteScriptRequest struct {
	Name       string `json:"name" binding:"required"`
	Operator   string `json:"operator" binding:"required"`
	OperatorID int64  `json:"operator_id" binding:"required"`
}
