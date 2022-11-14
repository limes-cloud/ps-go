package engine

import (
	"strings"
)

const (
	DefaultErrorCode          = "000400"
	SystemPanicErrorCode      = "110000"
	RunScriptErrorCode        = "110001"
	RunScriptFuncErrorCode    = "110002"
	ScriptFuncReturnErrorCode = "110003"
	ConditionErrorCode        = "110004"
	ModuleArgErrorCode        = "110005"
	RequestErrorCode          = "110006"
	NetworkErrorCode          = "110007"
	ActiveBreakErrorCode      = "110008"
	ActiveSuspendErrorCode    = "110009"
	BreakErrorCode            = "110010"
	SuspendErrorCode          = "110011"
)

type Error struct {
	Code string `json:"code"` //错误编码
	Msg  string `json:"msg"`  //错误描述
}

func (e *Error) Error() string {
	return e.Msg
}

// NewRunScriptError 执行脚本失败
func NewRunScriptError(msg string) error {
	return &Error{
		Code: RunScriptErrorCode,
		Msg:  msg,
	}
}

// NewRunScriptFuncError 执行函数失败
func NewRunScriptFuncError(msg string) error {
	return &Error{
		Code: RunScriptFuncErrorCode,
		Msg:  msg,
	}
}

// NewScriptFuncReturnError 函数返回值类型错误
func NewScriptFuncReturnError(msg string) error {
	return &Error{
		Code: ScriptFuncReturnErrorCode,
		Msg:  msg,
	}
}

// NewConditionError 准入条件表达式错误
func NewConditionError(msg string) error {
	return &Error{
		Code: ConditionErrorCode,
		Msg:  msg,
	}
}

// NewSystemPanicError 准入条件表达式错误
func NewSystemPanicError(msg string) error {
	return &Error{
		Code: SystemPanicErrorCode,
		Msg:  msg,
	}
}

// NewModuleArgError 调用module函数参数错误
func NewModuleArgError(msg string) error {
	return &Error{
		Code: ModuleArgErrorCode,
		Msg:  msg,
	}
}

// NewNetworkError 调用module函数参数错误
func NewNetworkError(msg string) error {
	return &Error{
		Code: NetworkErrorCode,
		Msg:  msg,
	}
}

// NewRequestError 发起请求报错
func NewRequestError(msg string) error {
	return &Error{
		Code: RequestErrorCode,
		Msg:  msg,
	}
}

// NewBreakError 错误中断错误
func NewBreakError(msg string) error {
	return &Error{
		Code: BreakErrorCode,
		Msg:  msg,
	}
}

// NewActiveBreakError 主动中断错误
func NewActiveBreakError(msg string) error {
	return &Error{
		Code: ActiveBreakErrorCode,
		Msg:  msg,
	}
}

// NewSuspendError 错误挂起
func NewSuspendError(msg string) error {
	return &Error{
		Code: SuspendErrorCode,
		Msg:  msg,
	}
}

// NewActiveSuspendError 主动中断错误
func NewActiveSuspendError(code, msg string) error {
	if strings.Trim(code, " ") == "" {
		code = ActiveSuspendErrorCode
	}
	return &Error{
		Code: code,
		Msg:  msg,
	}
}
