package handler

import (
	"fmt"
	"github.com/limeschool/gin"
	"ps-go/consts"
	"ps-go/engine"
	"ps-go/tools"
	"ps-go/tools/pool"
	"strings"
	"time"
)

func ProcessSchedule(ctx *gin.Context) {
	startTime := time.Now()

	trx := NewTrx()
	ctx.Writer.Header().Set(consts.ProcessScheduleTrx, trx)

	eg := engine.Get()

	// 获取调度规则
	path := ctx.Request.URL.Path
	path = strings.TrimLeft(path, consts.ApiPrefix)
	rule, err := eg.LoadRule(ctx, ctx.Request.Method, path)
	if err != nil {
		ctx.RespError(TransferError(err))
		return
	}

	// 校验参数
	requestInfo, err := eg.NewValidate(rule.Request).Bind(ctx)
	if err != nil {
		ctx.RespError(err)
		return
	}

	// 创建请求存储器
	runStore := eg.NewRunStore()
	runStore.SetData("request", requestInfo)

	// 创建运行器
	runner := eg.NewRunner(ctx, rule, runStore)
	runner.SetMethodAndPath(ctx.Request.Method, path)

	// 设置执行日志
	runner.NewLogger()
	runner.SetRequestLog(startTime, requestInfo)

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

func NewTrx() string {
	return fmt.Sprintf("TRX%v", tools.UUID())
}
