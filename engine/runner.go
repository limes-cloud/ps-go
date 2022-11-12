package engine

import (
	"fmt"
	"github.com/limeschool/gin"
	"go.uber.org/zap"
	"ps-go/errors"
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
	rule    *Rule  //当前执行的规则
	count   int    //总的执行步数
	index   int    //当前执行步数
	version string //执行流程的版本

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
		r.logger.SetStep(r.index + 1)

		// 获取当前执行的组件列表
		componentsCount := len(r.rule.Components[r.index])

		//当前没有需要执行的则直接跳过
		if componentsCount == 0 {
			r.index++
			continue
		}

		// 执行组件脚本/api
		r.RunComponent(componentsCount)

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
	log := r.logger.NewStepLog(r.index+1, count)
	defer log.SetRunTime(time.Now())

	// 设置需要执行的组件数量
	r.wg.Add(count)

	for i := 0; i < count; i++ {
		rt, err := r.NewRuntime(log, i)
		if err != nil {

			// 设置执行层错误
			log.SetError(err)

			r.err.Set(err)
			continue
		}
		_ = pool.Get().Invoke(rt)
	}

	r.wg.Wait()

}

func (r *runner) NewRuntime(log StepLog, action int) (*runtime, error) {
	com := r.rule.Components[r.index][action]
	return &runtime{
		stepLog:      log,
		wg:           r.wg,
		component:    com,
		response:     r.response,
		ctx:          r.ctx,
		step:         r.index,
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
	}
}

// WaitResponse 监听当前流程返回事件，只监听一次，不中断流程
func (r *runner) WaitResponse() {
	defer r.logger.SetResponseTime()

	if r.response.IsClose() {
		return
	}

	// 拿到了就删除返回通道，只能返回一次
	data, is := r.response.Get()
	if !is {
		return
	}
	defer r.response.Close()

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
	defer r.SetStatus(err)

	if r.err.IsClose() {
		return
	}

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
		r.response.Set(responseData{Code: e.Code, Msg: e.Msg})
		return
	}

	if e, ok := err.(*Error); ok {
		r.response.Set(responseData{Code: e.Code, Msg: e.Msg})
		return
	}

	r.response.Set(responseData{Code: errors.DefaultCode, Msg: err.Error()})
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

func (r *runner) Suspend(err error) {
	if !r.rule.Suspend {
		return
	}

	// 在设置了挂起的情况下，非中断错误，全部挂起
	if e, ok := err.(*Error); ok && e.Code != ActiveBreakErrorCode && e.Code != BreakErrorCode {
		// todo suspend
		fmt.Println("====suspend 挂起====")
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

// todo SaveLog
func (r *runner) SaveLog() {
	//r.logger
	fmt.Println(r.logger.GetString())
}

func (r *runner) SetRequestLog(t time.Time, data any) {
	r.logger.SetStartTime(t)
	r.logger.SetRequest(data)
	r.logger.SetVersion(r.version)
}

func (r *runner) SetError(err error) {
	r.logger.SetError(err)
	log := r.logger.GetStepErr(r.index)
	if log != nil {
		log.SetError(err)
	}
}
