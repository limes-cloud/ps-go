package errors

import "github.com/limeschool/gin"

var (
	ParamsError     = &gin.CustomError{Code: 100002, Msg: "参数验证失败"}
	AssignError     = &gin.CustomError{Code: 100003, Msg: "数据赋值失败"}
	DBError         = &gin.CustomError{Code: 100004, Msg: "数据库操作失败"}
	DBDupError      = &gin.CustomError{Code: 100005, Msg: "已存在重复的数据"}
	DBNotFoundError = &gin.CustomError{Code: 100006, Msg: "不存在数据"}

	RuleNotFoundError = &gin.CustomError{Code: 100100, Msg: "流程规则不存在"}
)
