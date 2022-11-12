package engine

import (
	"github.com/limeschool/gin"
	"go.uber.org/zap"
	"ps-go/errors"
	"ps-go/model"
	"ps-go/tools/pool"
	"sync"
	"time"
)

type Runner interface {
	Run()
	WaitResponse()
	WaitError()
	Response() any
	SaveLog()
	SetRequestLog(t time.Time, data any)
}

type runner struct {
	rule     *Rule  //当前执行的规则
	count    int    //总的执行层数
	index    int    //执行索引，控制流程
	curIndex int    //当前执行层数
	version  string //执行流程的版本
	trx      string //请求唯一表示
	method   string //请求方法
	path     string //请求路径

	wg       *sync.WaitGroup //运行时锁
	response *responseChan   //返回通道
	err      *errorChan      //错误通道
	ctx      *gin.Context    //上下文

	store    Store    //存储引擎
	runStore RunStore //存储运行时数据
	logger   Logger   //运行日志记录器
}

type responseData struct {
	Code any `json:"code"`
	Msg  any `json:"msg"`
	Data any `json:"data"`
}

func (r *runner) Run() {
	defer func() { // 防止意外Panic
		if p := recover(); p != nil {
			r.ctx.Log.Error("recover", zap.Any("panic", p))
		}
	}()

	for r.index < r.count {
		// 设置执行的步数
		r.logger.SetStep(r.curIndex + 1)

		// 获取当前执行的组件列表
		componentsCount := len(r.rule.Components[r.curIndex])

		//当前没有需要执行的则直接跳过
		if componentsCount == 0 {
			r.index++
			r.curIndex++
			continue
		}

		// 执行组件脚本/api
		r.RunComponent(componentsCount)
		r.curIndex++
		r.index++
	}

	// 释放通道
	r.err.Close()
	r.response.Close()

	// 存储日志
	r.logger.SetRunTime()
	r.SaveLog()
}

func (r *runner) RunComponent(count int) {
	log := r.logger.NewStepLog(r.curIndex+1, count)
	defer log.SetRunTime(time.Now())

	// 设置需要执行的组件数量
	r.wg.Add(count)

	for i := 0; i < count; i++ {
		rt, err := r.NewRuntime(log, i)
		if err != nil {
			// 设置执行层错误
			log.SetError(err)
			r.err.SetAndClose(err)
			continue
		}
		// 异常重启时，存在同一层执行成功的组件，不在执行
		if rt.isFinish {
			r.wg.Done()
			continue
		}
		_ = pool.Get().Invoke(rt)
	}

	r.wg.Wait()
}

func (r *runner) NewRuntime(log StepLog, action int) (*runtime, error) {
	com := r.rule.Components[r.curIndex][action]
	return &runtime{
		stepLog:      log,
		trx:          r.trx,
		wg:           r.wg,
		component:    com,
		response:     r.response,
		ctx:          r.ctx,
		step:         r.curIndex,
		action:       action,
		maxRetry:     com.RetryMaxCount,
		retryMaxWait: com.RetryMaxWait,
		store:        r.store,
		err:          r.err,
		runStore:     r.runStore,
	}, nil
}

func (r *runner) NewLogger() {
	r.logger = &runLog{
		lock:    sync.RWMutex{},
		LogId:   r.ctx.TraceID,
		Step:    r.count,
		Version: r.version,
		Trx:     r.trx,
	}
}

// WaitResponse 监听当前流程返回事件，只监听一次，不中断流程
func (r *runner) WaitResponse() {
	defer r.logger.SetResponseTime()

	// 拿到了就删除返回通道，只能返回一次
	data, is := r.response.Get()
	if !is {
		return
	}
	//defer r.response.Close()

	r.runStore.SetData("response", map[string]any{
		"body": map[string]any{
			"code": data.Code,
			"msg":  data.Msg,
			"data": data.Data,
		},
	})
}

// WaitError 监听当前流程错误事件，只监听一次，并且中断流程执行
func (r *runner) WaitError() {
	var err error

	// 设置请求状态
	defer func() {
		r.SetStatus(err)
	}()

	// 监听等待错误中断事件
	err, is := r.err.Get()
	if !is || err == nil {
		return
	}

	r.SetError(err)
	r.Suspend(err)

	//当遇到报错时，应该先处理完事物才done 否则无法准确中断流程执行。
	defer r.wg.Done()

	// 中断执行流程
	r.index = r.count

	// 处理返回值
	if !r.response.IsClose() {
		r.ResponseError(err)
	}
}

func (r *runner) ResponseError(err error) {
	if e, ok := err.(*gin.CustomError); ok {
		r.response.SetAndClose(responseData{Code: e.Code, Msg: e.Msg})
		return
	}

	if e, ok := err.(*Error); ok {
		r.response.SetAndClose(responseData{Code: e.Code, Msg: e.Msg})
		return
	}

	r.response.SetAndClose(responseData{Code: errors.DefaultCode, Msg: err.Error()})
}

func (r *runner) Response() any {
	var resp any
	body := r.rule.Response.Body
	if body != nil {
		resp = r.runStore.GetMatchData(r.rule.Response.Body)
	}
	// 设置返回的数据
	r.logger.SetResponse(resp)
	return resp
}

func (r *runner) GetRuleToString() string {
	// 由于getMatchData 修改了一些值，所以不能直接取挂载的rule
	rule, _ := r.store.LoadRule(r.ctx)
	str, _ := json.MarshalToString(rule)
	return str
}

func (r *runner) GetDataToString() string {
	str, _ := json.MarshalToString(r.runStore.GetAll())
	return str
}

func (r *runner) GetComponentErrorNames() string {
	log := r.logger.GetStepLog(r.curIndex)
	if log == nil {
		r.ctx.Log.Error("GetComponentErrorNames Error", zap.Any("index", r.curIndex))
		return "[]"
	}

	names := log.GetComponentErrorNames()
	str, _ := json.MarshalToString(names)
	return str
}

func (r *runner) Suspend(err error) {
	if !r.rule.Suspend {
		return
	}

	// 在设置了挂起的情况下，中断错误则直接返回
	if e, ok := err.(*Error); !ok || e.Code == ActiveBreakErrorCode || e.Code == BreakErrorCode {
		return
	}

	// 进行任务存库
	suspendLog := model.SuspendLog{
		Trx:          r.trx,
		LogID:        r.ctx.TraceID,
		Method:       r.method,
		Path:         r.path,
		Version:      r.version,
		Step:         r.count,
		CurStep:      r.curIndex + 1,
		ErrMsg:       err.Error(),
		Rule:         r.GetRuleToString(),
		Data:         r.GetDataToString(),
		ErrComponent: r.GetComponentErrorNames(),
	}
	if err = suspendLog.Create(r.ctx); err != nil {
		r.ctx.Log.Error("流程存储失败：%v", zap.Any("trx", r.trx), zap.Any("err", err))
	}
}

func (r *runner) SetStatus(err error) {
	if err == nil {
		r.logger.SetStatus(RunSuccess)
		return
	}

	r.logger.SetStatus(RunBreak)

	if e, ok := err.(*Error); ok {
		if e.Code == ActiveBreakErrorCode { // 主动中断
			r.logger.SetStatus(RunActiveBreak)
		}
		if e.Code == BreakErrorCode { //错误中断
			r.logger.SetStatus(RunBreak)
		}
		if r.rule.Suspend && e.Code == SuspendErrorCode { // 错误中断
			r.logger.SetStatus(RunSuspend)
		}
		if r.rule.Suspend && e.Code == ActiveSuspendErrorCode { //主动中断
			r.logger.SetStatus(RunActiveSuspend)
		}
	}
}

func (r *runner) SaveLog() {
	msg := r.logger.GetString()
	r.ctx.Log.Info("link log", zap.Any("", msg))

	if !r.rule.Record {
		return
	}

	// 存储请求链路
	log := model.RunLog{
		Trx:     r.trx,
		LogID:   r.ctx.TraceID,
		Method:  r.method,
		Path:    r.path,
		Version: r.version,
		Msg:     msg,
	}
	if err := log.Create(r.ctx); err != nil {
		r.ctx.Log.Error("执行流程存储失败：%v", zap.Any("trx", r.trx), zap.Any("err", err))
	}
}

func (r *runner) SetRequestLog(t time.Time, data any) {
	r.logger.SetStartTime(t)
	r.logger.SetRequest(data)
	r.logger.SetVersion(r.version)
}

func (r *runner) SetError(err error) {
	r.logger.SetError(err)
	log := r.logger.GetStepLog(r.curIndex)
	if log != nil {
		log.SetError(err)
	}
}
