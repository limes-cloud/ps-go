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
	RunActiveSuspend = "ActiveSuspend"
	RunErrorSuspend  = "ErrorSuspend"
	RunErrorBreak    = "ErrorBreak"
	RunActiveBreak   = "ActiveBreak"
	RunSuccess       = "Success"
)
