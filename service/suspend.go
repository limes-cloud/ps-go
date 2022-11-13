package service

import (
	json "github.com/json-iterator/go"
	"github.com/limeschool/gin"
	"ps-go/consts"
	"ps-go/engine"
	"ps-go/errors"
	"ps-go/model"
	"ps-go/tools/pool"
	"ps-go/types"
)

func GetSuspend(ctx *gin.Context, in *types.GetSuspendRequest) (model.SuspendLog, error) {
	var err error
	suspend := model.SuspendLog{}
	if in.ID != 0 {
		err = suspend.OneByID(ctx, in.ID)
	}

	if in.Trx != "" {
		err = suspend.OneByTrx(ctx, in.Trx)
	}

	return suspend, err
}

func PageSuspend(ctx *gin.Context, in *types.PageSuspendRequest) ([]model.SuspendLog, int64, error) {
	suspend := model.SuspendLog{}
	return suspend.Page(ctx, in.Page, in.Count, in)
}

func SuspendRecover(ctx *gin.Context, in *types.SuspendRecoverRequest) (any, error) {
	suspend := model.SuspendLog{}
	if err := suspend.OneByTrx(ctx, in.Trx); err != nil {
		return nil, err
	}

	// 通过trx获取对应日志信息
	log := model.RunLog{}
	if err := log.OneByTrx(ctx, suspend.Trx); err != nil {
		return nil, err
	}

	var data = make(map[string]any) // 执行上下文数据
	var rule engine.Rule            //执行规则
	var names []string              //所在层需要重试的组件名

	if err := json.UnmarshalFromString(suspend.Rule, &rule); err != nil {
		return nil, errors.NewF("任务重启失败，rule格式错误:%v", err.Error())
	}

	if err := json.UnmarshalFromString(suspend.Data, &data); err != nil {
		return nil, errors.NewF("任务重启失败，data格式错误:%v", err.Error())
	}

	if err := json.UnmarshalFromString(suspend.ErrComponent, &names); err != nil {
		return nil, errors.NewF("任务重启失败，err_component格式错误:%v", err.Error())
	}

	// 将重启的data参数载入
	for key, val := range in.Data {
		data[key] = val
	}

	// 重新唤起调度程序
	// 新建运行调度器
	eg := engine.Get()
	ctx.Writer.Header().Set(consts.ProcessScheduleTrx, suspend.Trx)

	// 创建存储器
	runStore := eg.NewRunStoreByData(data)

	// 创建运行器
	runner := eg.NewRunner(ctx, &rule, runStore)
	runner.SetMethodAndPath(suspend.Method, suspend.Path)
	runner.NewLoggerFromString(log.Msg)

	// 设置恢复重试从第几层开始
	runner.SetStep(suspend.CurStep - 1)
	if err := runner.SetStepComponentRetry(suspend.CurStep-1, names); err != nil {
		return nil, err
	}

	// 删除中断信息
	if err := suspend.DeleteByTrx(ctx, suspend.Trx); err != nil {
		return nil, err
	}

	// 执行服务
	_ = pool.Get().Invoke(runner)

	// 异步监听错误信息
	go runner.WaitError()
	// 同步等待返回结果
	runner.WaitResponse()

	// 获取返回结果
	return runner.Response(), nil
}

func UpdateSuspend(ctx *gin.Context, in *types.UpdateSuspendRequest) error {
	suspend := model.SuspendLog{}
	suspend.CurStep = in.CurStep

	if in.ErrNames != nil {
		suspend.ErrComponent, _ = json.MarshalToString(in.ErrNames)
	}

	if in.Data != nil {
		suspend.Data, _ = json.MarshalToString(in.Data)
	}

	if in.Rule != nil {
		// todo ruleCheck
		suspend.Rule, _ = json.MarshalToString(in.Rule)
	}

	return suspend.Update(ctx)
}
