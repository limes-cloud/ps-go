package engine

import (
	"fmt"
	"github.com/limeschool/gin"
	"github.com/robertkrimen/otto"
	"ps-go/consts"
	"ps-go/errors"
	"ps-go/tools/pool"
	"sync"
	"time"
)

type runtime struct {
	wg           *sync.WaitGroup
	component    Component
	runStore     *runStore
	ctx          *gin.Context
	retry        int //重试次数
	maxRetry     int //最大重试次数
	retryMaxWait int //重试最大等待时长
	action       int
	step         int
	store        *store
	response     *responseChan
	err          *errorChan
	errType      int
}

func (r *runtime) Run() {

	var resp any
	var err error
	if r.component.Type == ComponentTypeApi {
		resp, err = r.runApi()
	} else {
		resp, err = r.runScript()
	}

	//处理请求异常
	if err != nil {
		if r.retry < r.maxRetry && r.errType != RunScriptError {
			if r.retryMaxWait != 0 {
				time.Sleep(r.waitTime(r.retry, r.maxRetry, r.retryMaxWait))
			}
			_ = pool.Get().Invoke(r)
			r.retry++
		} else {
			r.err.Set(err)
		}

		return
	}
	// 顺利执行完成
	r.wg.Done()
	r.runStore.SetData(r.component.OutputName, resp)
}

func (r *runtime) runApi() (any, error) {
	com := r.component

	var header = make(map[string]string)
	var body any
	var auth = make([]string, 0)

	// 转换header
	if len(com.Header) != 0 {
		headerData := r.runStore.GetMatchData(com.Header)

		if data, ok := headerData.(map[string]string); ok {
			header = data
		}

		if data, ok := headerData.(map[string]any); ok {
			for key, val := range data {
				header[key] = fmt.Sprint(val)
			}
		}
	}

	// 转换body
	if len(com.Input) != 0 {
		body = r.runStore.GetMatchData(com.Input)
	}

	// 转换auth
	if len(com.Auth) != 0 {
		authData := r.runStore.GetMatchData(com.Auth)
		if data, ok := authData.([]string); ok {
			auth = data
		}
		if data, ok := authData.([]any); ok {
			for key, val := range data {
				auth[key] = fmt.Sprint(val)
			}
		}
	}

	request := HttpRequest{
		url:         com.Url,
		method:      com.Method,
		header:      header,
		auth:        auth,
		body:        body,
		contentType: com.ContentType,
		timeout:     com.Timeout,
		respType:    com.RespType,
	}

	return request.Do()
}

func (r *runtime) runScript() (any, error) {
	script, err := r.store.LoadScript(r.ctx, r.component.Url)
	if err != nil {
		return nil, err
	}
	vm := otto.New()
	// 超时设置
	go func() {
		// 默认60秒
		if r.component.Timeout == 0 {
			r.component.Timeout = 60
		}

		time.Sleep(time.Duration(r.component.Timeout) * time.Second) // Stop after two seconds

		vm.Interrupt <- func() {
			panic("script run timeout")
		}

	}()

	if _, err = vm.Run(script); err != nil {
		r.errType = RunScriptError
		return nil, errors.NewF("执行脚本失败：%v", err.Error())
	}

	value, err := vm.Call(consts.ProcessScheduleFunc, nil)
	if err != nil {
		r.errType = RunScriptError
		return nil, errors.NewF("调用脚本函数失败：%v", err.Error())
	}

	if v, err := value.Export(); err != nil {
		r.errType = RunScriptError
		return nil, errors.NewF("函数返回值类型错误")
	} else {
		return v, nil
	}
}

func (r *runtime) waitTime(cur, max, wait int) time.Duration {
	if wait == 0 {
		wait = 10
	}

	if max > 5 || max < 0 {
		max = 5
	}

	if cur >= max {
		return time.Duration(wait) * time.Second
	} else {
		return time.Duration((wait/max)*cur) * time.Second
	}
}
