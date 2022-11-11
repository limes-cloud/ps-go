package engine

import "github.com/json-iterator/go"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	Int    = "int"
	Float  = "float"
	String = "string"
	Slice  = "slice"
	Bool   = "bool"
	Map    = "object"

	ComponentTypeApi    = "api"
	ComponentTypeScript = "script"

	LogDatetimeFormat = "2006-01-02 15:04:05.000"
)

const (
	RunActiveSuspend = "主动挂起"
	RunErrorSuspend  = "错误挂起"
	RunBreak         = "流程中断"
	RunStatus        = "流程结束"
)

const (
	RunScriptError = 1 + iota // 执行脚本错误
	NetworkError   = 1 + iota // 网络错误
)
