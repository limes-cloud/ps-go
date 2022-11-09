package engine

import (
	"fmt"
	"github.com/limeschool/gin"
	"github.com/robertkrimen/otto"
	"ps-go/errors"
	"ps-go/tools/pool"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

type Runner interface {
	Run()
	WaitResponse()
	WaitError()
	Response() any
}

type runner struct {
	rule     *Rule           //当前执行的规则
	count    int             //总的执行步数
	index    int             //当前执行步数
	runStore RunStore        //存储运行时数据
	wg       *sync.WaitGroup //运行时锁
	store    *store          //存储引擎
	response *responseChan   //返回通道
	err      *errorChan      //错误通道
	ctx      *gin.Context    //上下文
}

type responseData struct {
	Code any `json:"code"`
	Msg  any `json:"msg"`
	Data any `json:"data"`
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
