package engine

const (
	Int    = "int"
	Float  = "float"
	String = "string"
	Slice  = "slice"
	Bool   = "bool"
	Map    = "object"

	ComponentTypeApi    = "api"
	ComponentTypeScript = "script"
	LogDatetimeFormat   = "2006-01-02 15:04:05.000"
)

const (
	RunActiveSuspend = "主动挂起" //ActiveSuspend
	RunSuspend       = "错误挂起" //ErrorSuspend
	RunBreak         = "错误中断" //ErrorBreak
	RunActiveBreak   = "主动中断" //ActiveBreak
	RunSuccess       = "成功执行" //Success
)
