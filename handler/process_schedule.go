package handler

import (
	"github.com/limeschool/gin"
	"ps-go/engine"
	"ps-go/tools/pool"
)

func ProcessSchedule(ctx *gin.Context) {
	eg := engine.Get()

	// 获取调度规则
	rule, err := eg.LoadRule(ctx)
	if err != nil {
		ctx.RespError(err)
		return
	}

	// 校验参数
	vp, err := eg.NewValidate(rule.Request).Bind(ctx)
	if err != nil {
		ctx.RespError(err)
		return
	}

	// 创建运行器
	runner := eg.NewRunner(ctx, rule, vp)

	// 执行服务
	_ = pool.Get().Invoke(runner)
	// 异步监听错误信息
	go runner.WaitError()
	// 同步等待返回结果
	runner.WaitResponse()
	// 获取返回结果
	data := runner.Response()
	// 输出
	ctx.RespJson(data)
}
