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
)

const (
	RunScriptError = 1 + iota // 执行脚本错误
	NetworkError   = 1 + iota // 网络错误

)
