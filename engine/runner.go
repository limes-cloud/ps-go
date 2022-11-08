package engine

import (
	"fmt"
	"github.com/limeschool/gin"
	"github.com/robertkrimen/otto"
	"github.com/spf13/viper"
	"ps-go/errors"
	"ps-go/tools/pool"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

type runStore struct {
	data *viper.Viper
}

// SetData 递归设置数据
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

// GetData 直接通过viper 获取数据
func (r *runStore) GetData(key string) any {
	return r.data.Get(key)
}

// GetMatchData 获取存在表达式的数据
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

			entry, err := r.IsEntry(i)
			if err != nil {
				r.err.Set(err)
				continue
			}

			if !entry {
				r.wg.Done()
				continue
			}

			rt, err := r.NewRuntime(i)
			if err != nil {
				r.err.Set(err)
				continue
			}

			_ = pool.Get().Invoke(rt)
		}

		r.wg.Wait()
		r.index++
	}

	// 释放通道
	r.err.Close()
	r.response.Close()

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

// IsEntry 是否准入
func (r *runner) IsEntry(action int) (bool, error) {
	com := r.rule.Components[r.index][action]
	if com.Condition == "" {
		return true, nil
	}

	// 需要执行表达式
	reg := regexp.MustCompile(`\{(\w|\.)+\}`)
	variable := reg.FindAllString(com.Condition, -1)

	script := ""
	cond := com.Condition

	for index, valIndex := range variable {
		key := fmt.Sprintf("a_%v", index)

		cond = strings.ReplaceAll(cond, valIndex, key)

		newVal := r.runStore.GetData(valIndex[1 : len(valIndex)-1])
		if newVal == nil {
			script += fmt.Sprintf("let %v = null;", key)
			continue
		}

		// 进行变量转换
		switch newVal.(type) {
		case uint8, uint16, uint32, uint, uint64, int8, int16, int32, int, int64, float64, float32, bool:
			script += fmt.Sprintf("let %v = %v;", key, fmt.Sprint(newVal))

		case string:
			script += fmt.Sprintf(`let %v = "%v";`, key, newVal.(string))

		case []any, map[string]any:
			str, _ := json.MarshalToString(newVal)
			script += fmt.Sprintf(`let %v = %v;`, key, str)

		default:
			tp := reflect.TypeOf(newVal)

			if tp.Kind() == reflect.Map || tp.Kind() == reflect.Slice {
				str, _ := json.MarshalToString(newVal)
				script += fmt.Sprintf(`let %v = %v;`, key, str)
			} else {
				//处理不了的数据值默认为 undefined
				script += fmt.Sprintf("let %v = undefined;", key)
			}
		}

	}

	vm := otto.New()
	script = fmt.Sprintf("function condition(){%v return %v}", script, cond)
	_, err := vm.Run(script)
	if err != nil {
		return false, errors.NewF("准入表达式错误：%v", err.Error())
	}

	condVal, err := vm.Call("condition", nil)
	if err != nil {
		return false, errors.NewF("准入表达式执行错误：%v", err.Error())
	}
	if !condVal.IsBoolean() {
		return false, errors.NewF("准入表达式结果必须是bool")
	}
	return condVal.ToBoolean()
}

// WaitResponse 监听当前流程返回事件，只监听一次，不中断流程
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

// WaitError 监听当前流程错误事件，只监听一次，并且中断流程执行
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
	r.index = r.count

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
