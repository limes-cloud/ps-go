package types

type GetSecretRequest struct {
	ID   int64  `json:"id" form:"id"`
	Name string `json:"name" form:"name"`
}

type PageSecretRequest struct {
	Page  int `json:"page" form:"page" binding:"required" sql:"-"`
	Count int `json:"count" form:"count"  binding:"required,max=50"  sql:"-"`

	Name  string `json:"name" form:"name"`
	Start int64  `json:"start" form:"start" sql:"> ?" field:"created_at"`
	End   int64  `json:"end" form:"end" sql:"< ?" field:"created_at"`
}

type AddSecretRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"  binding:"required"`
	Context     string `json:"context"  binding:"required"`
	Operator    string `json:"operator" binding:"required"`
	OperatorID  int64  `json:"operator_id" binding:"required"`
}

type UpdateSecretRequest struct {
	ID          int64  `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"  binding:"required"`
	Context     string `json:"context"  binding:"required"`
	Operator    string `json:"operator" binding:"required"`
	OperatorID  int64  `json:"operator_id" binding:"required"`
}

type DeleteSecretRequest struct {
	ID         int64  `json:"id" binding:"required"`
	Operator   string `json:"operator" binding:"required"`
	OperatorID int64  `json:"operator_id" binding:"required"`
}
