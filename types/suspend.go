package types

type GetSuspendRequest struct {
	ID  int64  `json:"id" form:"id"`
	Trx string `json:"trx" form:"trx"`
}

type PageSuspendRequest struct {
	Page  int `json:"page" form:"page" binding:"required" sql:"-"`
	Count int `json:"count" form:"count"  binding:"required,max=50"  sql:"-"`

	Trx     string `json:"trx" form:"trx"`
	Method  string `json:"method" form:"method"`
	Path    string `json:"path" form:"path"`
	Version string `json:"version" form:"version"`
	LogID   string `json:"log_id" form:"log_id"`
	Start   int64  `json:"start" form:"start" sql:"> ?" field:"created_at"`
	End     int64  `json:"end" form:"end" sql:"< ?" field:"created_at"`
}

type SuspendRecoverRequest struct {
	Trx  string         `json:"trx"`
	Data map[string]any `json:"data"`
}

type UpdateSuspendRequest struct {
	ID       int64          `json:"id" form:"id"  binding:"required" `
	Data     map[string]any `json:"data"`
	Rule     map[string]any `json:"rule"`
	CurStep  int            `json:"cur_step"`
	ErrNames []string       `json:"err_names"`
}
