package engine

import (
	"fmt"
	"github.com/limeschool/gin"
	"github.com/spf13/viper"
	"ps-go/tools/pool"
	"reflect"
	"regexp"
	"sync"
)

type runStore struct {
	data *viper.Viper
}

func (r *runStore) SetData(key string, val any) {
	tp := reflect.ValueOf(val)
	if tp.Kind() == reflect.Map {
		iter := tp.MapRange()
		for iter.Next() {
			r.SetData(fmt.Sprintf("%v.%v", key, iter.Key()), iter.Value().Interface())
		}
	} else {
		r.data.Set(key, val)
	}
}

// GetMatchData 获取匹配之后的数据
func (r *runStore) GetMatchData(m any) any {
	reg := regexp.MustCompile(`\{(\w|\.)+\}`)

	switch m.(type) {
	case []any:
		var resp = m.([]any)
		for key, _ := range resp {
			resp[key] = r.GetMatchData(resp[key])
		}
		return resp

	case string:
		if str := reg.FindString(m.(string)); str != "" {
			return r.data.Get(str[1 : len(str)-1])
		}

	case map[string]any:
		var resp = m.(map[string]any)
		for key, _ := range resp {
			resp[key] = r.GetMatchData(resp[key])
		}
		return resp
	}

	return m
}

type responseChan struct {
	response chan responseData
	isClose  bool
	lock     sync.RWMutex
}

func (r *responseChan) Close() {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.isClose {
		return
	}
	r.isClose = true
	close(r.response)
}

func (r *responseChan) Set(data responseData) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.isClose {
		return
	}
	r.response <- data
}

func (r *responseChan) Get() (responseData, bool) {

	if r.isClose {
		return responseData{}, false
	}
	data, is := <-r.response
	return data, is
}

func (r *responseChan) IsClose() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.isClose
}

type errorChan struct {
	err     chan error
	isClose bool
	lock    sync.RWMutex
}

func (r *errorChan) Close() {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.isClose {
		return
	}
	r.isClose = true
	close(r.err)
}

func (r *errorChan) Set(data error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.isClose {
		return
	}
	r.err <- data
}

func (r *errorChan) Get() (error, bool) {
	if r.isClose {
		return nil, false
	}
	err, is := <-r.err
	return err, is
}

func (r *errorChan) IsClose() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.isClose
}

type runner struct {
	rule     *Rule           //当前执行的规则
	count    int             //总的执行步数
	index    int             //当前执行步数
	runStore *runStore       //存储运行时数据
	wg       *sync.WaitGroup //运行时锁
	store    *store          //存储引擎
	response *responseChan
	err      *errorChan
	ctx      *gin.Context
}

type responseData struct {
	Code any `json:"code"`
	Msg  any `json:"msg"`
	Data any `json:"data"`
}

type Runner interface {
	Run()
	WaitResponse()
	WaitError()
	Response() any
}

func (r *runner) Run() {
	for r.index < r.count {
		if r.index == -1 { //由于异常中断
			break
		}
		// 获取当前执行的组件列表
		componentsCount := len(r.rule.Components[r.index])

		//当前没有需要执行的则直接跳过
		if componentsCount == 0 {
			r.index++
			continue
		}

		// 设置需要执行的组件数量
		r.wg.Add(componentsCount)

		for i := 0; i < componentsCount; i++ {
			rt, err := r.NewRuntime(i)

			//处理返回错误数据
			if err != nil {
				r.err.Set(err)
				r.wg.Done()
			}

			_ = pool.Get().Invoke(rt)
		}

		r.wg.Wait()
		r.index++
	}

	// 释放通道
	r.err.Close()
	// 执行完所有流程释放通道
	if r.index != -1 {
		r.response.Close()
	}
}

func (r *runner) NewRuntime(action int) (*runtime, error) {
	com := r.rule.Components[r.index][action]
	return &runtime{
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

func (r *runner) WaitResponse() {
	if r.response.IsClose() {
		return
	}

	// 拿到了就删除返回通道，只能返回一次
	data, is := r.response.Get()
	if !is {
		return
	}
	defer r.response.Close()

	r.runStore.SetData("response.body.code", data.Code)
	r.runStore.SetData("response.body.data", data.Data)
	r.runStore.SetData("response.body.msg", data.Msg)
}

func (r *runner) WaitError() {

	if r.err.IsClose() {
		return
	}

	// 监听等待错误中断事件
	err, is := r.err.Get()
	if !is || err == nil {
		return
	}

	//当遇到报错时，应该先处理完事物才done 否则无法准确中断流程执行。
	defer r.wg.Done()

	// 中断执行流程
	r.index = -1

	// 处理返回值
	if !r.response.IsClose() {
		if customErr, ok := err.(*gin.CustomError); ok {
			r.response.Set(responseData{
				Code: customErr.Code,
				Msg:  customErr.Msg,
			})
		} else {
			r.response.Set(responseData{
				Code: 10000,
				Msg:  err.Error(),
			})
		}
	}

}

func (r *runner) Response() any {
	resp := r.rule.Response.Body
	if resp != nil {
		return r.runStore.GetMatchData(r.rule.Response.Body)
	}
	return nil
}
