package consts

const (
	ApiPrefix           = "/ps"
	GoRoutineCount      = 100_000 //最大的协程池数量
	GoRoutineExecSecond = 60      //最大执行时间60s
)

const (
	ProcessScheduleDB    = "ps"
	ProcessScheduleCache = "ps_cache"
	ProcessScheduleFunc  = "handler"
)

const (
	RespXml  = "xml"
	RespJson = "json"
	RespText = "text"
)
