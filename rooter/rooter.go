package rooter

import (
	"github.com/limeschool/gin"
	"ps-go/consts"
	"ps-go/handler"
)

// Init
//
//	@Description:初始化系统的api规则
//	@return *gin.Engine
func Init() *gin.Engine {
	engine := gin.Default()

	// 健康检查
	engine.GET("/check_healthy", gin.Success())

	// 调度引擎的后台服务api
	api := engine.Group("/api/v1")
	{
		api.GET("/rule", handler.GetRule)
		api.GET("/rule/page", handler.PageRule)
		api.POST("/rule", handler.AddRule)
		api.PUT("/rule", handler.UpdateRule)
		api.DELETE("/rule", handler.DeleteRule)

		api.GET("/script", handler.GetScript)
		api.GET("/script/page", handler.PageScript)
		api.POST("/script", handler.AddScript)
		api.PUT("/script", handler.UpdateScript)
		api.DELETE("/script", handler.DeleteScript)
	}

	// 提供给通用的调度入口 http://ps-go/ps/[rule_name]
	engine.Any(consts.ApiPrefix+"/*uri", handler.ProcessSchedule)

	return engine
}
