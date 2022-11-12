package consts

const (
	ApiPrefix            = "/ps"
	GoRoutineCount       = 100_000 //最大的协程池数量
	GoRoutineExecSecond  = 62      //最大执行时间62s
	ComponentExecSecond  = 60      //组件最大执行时间60s
	ProcessScheduleTrx   = "Trx"
	ProcessScheduleDB    = "ps"       //数据库
	ProcessScheduleCache = "ps_cache" //缓存所用的redis
	ProcessScheduleFunc  = "handler"  //默认的调度处理函数
	ProcessScheduleLock  = "ps_cache" //分布式锁所用的redis
	RuleHistoryCount     = 3          //rule最大的历史版本数量
	ScriptHistoryCount   = 3          //script最大的历史版本数量
	MaxLogReplicaCount   = 32         //运行日志表最大的副本数量
)

const ()

const (
	RespXml  = "xml"
	RespJson = "json"
	RespText = "text"
)
