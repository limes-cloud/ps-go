package engine

import (
	"fmt"
	"github.com/limeschool/gin"
	"github.com/robertkrimen/otto"
	"ps-go/consts"
	"ps-go/errors"
	"ps-go/tools"
	"ps-go/tools/pool"
	"sync"
	"time"
)

type runtime struct {
	vm           *otto.Otto      // js 运行虚拟器
	wg           *sync.WaitGroup // 运行时锁，与runner共用一个锁
	component    Component       // 运行组件信息
	runStore     RunStore        // 运行存储器
	ctx          *gin.Context    // 上下文
	retry        int             // 重试次数
	maxRetry     int             // 最大重试次数
	retryMaxWait int             // 重试最大等待时长
	action       int             // 当前所在步数
	step         int             // 当前所在层级
	store        *store          // 全局存储器
	response     *responseChan   // 返回通道
	err          *errorChan      // 错误通道
	errType      int             // 错误类型
}

func (r *runtime) Run() {

	var resp any
	var err error

	// 进行变量参数转换
	r.transferData()

	//判断是否使用缓存
	cache := r.newRunCache()
	if r.component.IsCache {
		if resp, err = cache.getCache(); err == nil {

			r.runStore.SetData(r.component.OutputName, resp)
			r.wg.Done()

			return
		}
	}

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
	r.runStore.SetData(r.component.OutputName, resp)
	if r.component.IsCache {
		cache.setCache(resp)
	}
	r.wg.Done()
}

func (r *runtime) newRunCache() *runCache {
	return &runCache{
		r,
	}
}

func (r *runtime) runApi() (any, error) {
	com := r.component

	var header = make(map[string]string)
	var body any
	var auth = make([]string, 0)

	// 转换header
	if len(com.Header) != 0 {
		for key, val := range com.Header {
			header[key] = fmt.Sprint(val)
		}
	}

	// 转换body
	if len(com.Input) != 0 {
		body = com.Input
	}

	// 转换auth
	if len(com.Auth) != 0 {
		for key, val := range com.Auth {
			auth[key] = fmt.Sprint(val)
		}
	}

	request := tools.HttpRequest{
		Url:          com.Url,
		Method:       com.Method,
		Header:       header,
		Auth:         auth,
		Body:         body,
		ContentType:  com.ContentType,
		Timeout:      com.Timeout,
		ResponseType: com.ResponseType,
		DataType:     com.DataType,
	}

	return request.Result()
}

// transferData 对可输入变量字段进行转换
func (r *runtime) transferData() {
	if len(r.component.Header) != 0 {
		r.runStore.GetMatchData(r.component.Header)
	}

	if len(r.component.Input) != 0 {
		r.runStore.GetMatchData(r.component.Input)
	}

	if len(r.component.Auth) != 0 {
		r.runStore.GetMatchData(r.component.Auth)
	}

}

func (r *runtime) runScript() (resp any, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = errors.NewF("调用脚本函数失败:%v", p)
		}
	}()

	script, err := r.store.LoadScript(r.ctx, r.component.Url)
	if err != nil {
		return nil, err
	}

	// 创建调用js vm
	r.vm = otto.New()

	// 超时设置
	go func() {
		// 默认60秒
		if r.component.Timeout == 0 {
			r.component.Timeout = 60
		}

		time.Sleep(time.Duration(r.component.Timeout) * time.Second) // Stop after two seconds

		r.vm.Interrupt <- func() {
			panic("script run timeout")
		}

	}()

	if _, err = r.vm.Run(script); err != nil {
		r.errType = RunScriptError
		return nil, errors.NewF("执行脚本失败：%v", err.Error())
	}

	// 获取调用入参
	ctx := GetGlobalJsModule(r)
	input := r.component.Input

	// 调用执行
	value, err := r.vm.Call(consts.ProcessScheduleFunc, r.ctx, ctx, input)
	if err != nil {
		r.errType = RunScriptError
		return nil, errors.NewF("调用脚本函数失败：%v", err.Error())
	}

	// 返回结果
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
