package rooter

import (
	"github.com/limeschool/gin"
	"ps-go/consts"
	"ps-go/handler"
)

//	Init @Description:初始化系统的api规则
func Init() *gin.Engine {
	engine := gin.Default()

	// 健康检查
	engine.GET("/check_healthy", gin.Success())

	// 调度引擎的后台服务api
	api := engine.Group("/api/v1")
	{
		// 流程规则相关api
		api.GET("/rule", handler.GetRule)
		api.GET("/rule/page", handler.PageRule)
		api.POST("/rule", handler.AddRule)
		api.PUT("/rule/switch_version", handler.UpdateRule)
		api.DELETE("/rule", handler.DeleteRule)

		// 脚本相关api
		api.GET("/script", handler.GetScript)
		api.GET("/script/page", handler.PageScript)
		api.POST("/script", handler.AddScript)
		api.PUT("/script/switch_version", handler.SwitchVersion)
		api.DELETE("/script", handler.DeleteScript)

		// 异常中断api
		api.GET("/suspend/page", handler.PageSuspend)
		api.GET("/suspend", handler.GetSuspend)
		api.POST("/suspend/recover", handler.SuspendRecover)
		api.PUT("/suspend", handler.UpdateSuspend)

		// 执行日志相关api
		api.GET("/run_log", handler.GetRunLog)
	}

	// 提供给通用的调度入口 http://ps-go/ps/[rule_name]
	engine.Any(consts.ApiPrefix+"/*uri", handler.ProcessSchedule)

	return engine
}
