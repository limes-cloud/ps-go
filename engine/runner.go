package engine

import (
	"github.com/limeschool/gin"
	"go.uber.org/zap"
	"ps-go/errors"
	"ps-go/model"
	"ps-go/tools"
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
	NewLogger()
	NewLoggerFromString(str string)
	SetStep(index int)
	SetMethodAndPath(m, p string)
	SetStepComponentRetry(index int, names []string) error
}

type runner struct {
	rule     *Rule  //当前执行的规则
	copyRule *Rule  //规则的副本，异常恢复时存档
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
			log.SetError(err) // 设置执行层错误
			r.err.SetAndClose(err, r.wg)
			continue
		}
		// 异常重启时，存在同一层执行成功的组件，不在执行
		if rt.component.IsFinish {
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

func (r *runner) NewLoggerFromString(str string) {
	log := &runLog{}
	if json.UnmarshalFromString(str, log) != nil {
		log.start, _ = time.Parse(LogDatetimeFormat, log.StartDatetime)
		log.CurStep = log.CurStep - 1
	}
	r.logger = log
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

	r.runStore.SetData("response", map[string]any{"body": data})
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

// SetStep 设置流程当前的层数
func (r *runner) SetStep(index int) {
	r.index = index
}

// SetMethodAndPath 设置组件的请求方法以及path
func (r *runner) SetMethodAndPath(m, p string) {
	r.path = p
	r.method = m
}

// SetStepComponentRetry 设置需要重新执行的组件
func (r *runner) SetStepComponentRetry(index int, names []string) error {
	if len(r.rule.Components) <= index {
		return errors.New("重试索引值大于组件层数")
	}
	for key, com := range r.rule.Components[index] {
		if !tools.InList(names, com.Name) {
			r.rule.Components[index][key].IsFinish = true
		}
	}
	return nil
}

// ResponseError 错误信息分类发送到返回器
func (r *runner) ResponseError(err error) {
	if e, ok := err.(*gin.CustomError); ok {
		r.response.SetAndClose(map[string]any{"code": e.Code, "msg": e.Msg})
		return
	}

	if e, ok := err.(*Error); ok {
		r.response.SetAndClose(map[string]any{"code": e.Code, "msg": e.Msg})
		return
	}

	r.response.SetAndClose(map[string]any{"code": errors.DefaultCode, "msg": err.Error()})
}

// Response 进行数据返回
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

// GetRuleToString 获取规则
func (r *runner) GetRuleToString() string {
	str, _ := json.MarshalToString(r.copyRule)
	return str
}

// GetDataToString 获取上下文数据
func (r *runner) GetDataToString() string {
	str, _ := json.MarshalToString(r.runStore.GetAll())
	return str
}

// GetComponentErrorNames 获取挂起时错误的组件名称
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

// Suspend 服务挂起
func (r *runner) Suspend(err error) {
	if !r.rule.Suspend {
		return
	}

	// 在设置了挂起的情况下，中断错误则直接返回
	var code = DefaultErrorCode
	if e, ok := err.(*Error); ok {
		if e.Code == ActiveBreakErrorCode || e.Code == BreakErrorCode {
			return
		}
		code = e.Code
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
		ErrCode:      code,
		ErrMsg:       err.Error(),
		Rule:         r.GetRuleToString(),
		Data:         r.GetDataToString(),
		ErrComponent: r.GetComponentErrorNames(),
	}
	if err = suspendLog.Create(r.ctx); err != nil {
		r.ctx.Log.Error("流程存储失败：%v", zap.Any("trx", r.trx), zap.Any("err", err))
	}
}

// SetStatus 设置执行状态
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

// SaveLog 存储请求链日志
func (r *runner) SaveLog() {

	msg := r.logger.Get()
	r.ctx.Log.Info("link log", zap.Any("data", msg))

	if !r.rule.Record {
		return
	}

	msgStr, _ := json.MarshalToString(msg)
	// 存储请求链路
	log := model.RunLog{
		Trx:     r.trx,
		LogID:   r.ctx.TraceID,
		Method:  r.method,
		Path:    r.path,
		Version: r.version,
		Msg:     msgStr,
		Step:    r.count,
		CurStep: r.curIndex + 1,
		Status:  r.logger.GetStatus(),
	}

	if err := log.Create(r.ctx); err != nil {
		r.ctx.Log.Error("执行流程存储失败：%v", zap.Any("trx", r.trx), zap.Any("err", err))
	}
}

// SetRequestLog 设置执行开始请求数据
func (r *runner) SetRequestLog(t time.Time, data any) {
	r.logger.SetStartTime(t)
	r.logger.SetRequest(data)
	r.logger.SetVersion(r.version)
}

// SetError 设置流程执行error原因
func (r *runner) SetError(err error) {
	r.logger.SetError(err)
	log := r.logger.GetStepLog(r.curIndex)
	if log != nil {
		log.SetError(err)
	}
}
