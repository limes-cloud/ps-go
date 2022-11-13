package engine

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/limeschool/gin"
	"github.com/robertkrimen/otto"
	"go.uber.org/zap"
	"ps-go/consts"
	"ps-go/tools"
	"ps-go/tools/lock"
	"time"
)

// GetGlobalJsModule 全局module 函数
func GetGlobalJsModule(r *runtime) any {
	return gin.H{
		"request": RequestModule(r),       //发送请求
		"log":     LogModule(r),           //打印日志
		"data":    StoreModule(r),         //存储数据
		"logId":   LogIDModule(r),         //获取logId
		"trx":     TrxModule(r),           //获取trx
		"suspend": ActiveSuspendModule(r), //主动挂起
		"break":   ActiveBreakModule(r),   //主动中断
	}
}

type requestArg struct {
	Url          string            `json:"url"`          //请求的url
	Method       string            `json:"method"`       //请求的方法
	Body         any               `json:"body"`         //请求的body
	Header       map[string]string `json:"header"`       //请求的header
	Auth         []string          `json:"auth"`         //请求的auth
	ContentType  string            `json:"contentType"`  //请求类型
	DataType     string            `json:"dataType"`     //数据类型
	Timeout      int               `json:"timeout"`      //超时时间
	ResponseType string            `json:"responseType"` //返回类型
	IsCache      bool              `json:"isCache"`      //是否缓存
	OnlyData     *bool             `json:"onlyData"`     //是否只返回data,不携带header等 默认true
}

// RequestModule 设置http 请求函数，返回详细请求信息包括header头
func RequestModule(r *runtime) func(call otto.FunctionCall) otto.Value {

	// 解析请求参数
	handleParseArg := func(call otto.FunctionCall) *requestArg {
		if len(call.ArgumentList) == 0 {
			panic(NewModuleArgError("request method argument not null"))
		}

		param, err := call.Argument(0).Export()
		if err != nil {
			panic(NewModuleArgError(fmt.Sprintf("request method argument type err:%v", err.Error())))
		}

		byteData, err := json.Marshal(param)
		if err != nil {
			panic(NewModuleArgError(fmt.Sprint("request method argument must object")))
		}

		arg := &requestArg{}
		if err = json.Unmarshal(byteData, arg); err != nil {
			panic(NewModuleArgError(fmt.Sprintf("request method argument type err:%v", err.Error())))
		}

		if arg.OnlyData == nil {
			arg.OnlyData = tools.Bool(true)
		}

		return arg
	}

	// 发起请求
	handleRequest := func(arg *requestArg) *tools.HttpRequest {
		var err error
		// 创建请求日志
		log := r.componentLog.NewRequestLog()

		request := tools.HttpRequest{
			Url:          arg.Url,
			Method:       arg.Method,
			Body:         arg.Body,
			Header:       arg.Header,
			Auth:         arg.Auth,
			ContentType:  arg.ContentType,
			DataType:     arg.DataType,
			Timeout:      arg.Timeout,
			ResponseType: arg.ResponseType,
		}

		// 设置请求参数
		log.SetRequest(request)

		if err = request.Do(); err != nil {
			if _, ok := err.(*gin.CustomError); ok {
				err = NewRequestError(err.Error())
			} else {
				err = NewNetworkError(err.Error())
			}
			log.SetError(err)
			panic(err)
		}

		// 设置返回结果
		log.SetRespBody(request.ResponseBody())
		log.SetRespCode(request.ResponseCode())
		log.SetRespHeader(request.ResponseHeader())
		log.SetRespCookies(request.ResponseCookies())

		return &request
	}

	// 获取缓存
	handleGetCache := func(client *redis.Client, key string) (otto.Value, bool) {
		// 查询redis缓存
		if str, err := client.Get(context.TODO(), key).Result(); err == nil && str != "" {
			resp := map[string]any{}
			if json.UnmarshalFromString(str, &resp) == nil {
				value, _ := r.vm.ToValue(resp)
				return value, true
			}
		}
		return otto.Value{}, false
	}

	// 获取缓存的key
	getCacheKey := func(data any) []byte {
		b, _ := json.Marshal(data)
		return b
	}

	// 导出函数
	return func(call otto.FunctionCall) otto.Value {
		arg := handleParseArg(call)

		client := r.ctx.Redis(consts.ProcessScheduleCache)
		cacheKey := ""
		// 开启了缓存，则查询缓存
		if arg.IsCache {
			byteData := getCacheKey(arg)
			cacheKey = fmt.Sprintf("request_%x", md5.Sum(byteData))

			// 查询缓存
			if value, ok := handleGetCache(client, cacheKey); ok {
				return value
			}

			// 上锁
			lc := lock.NewLock(r.ctx, fmt.Sprintf("lock_%v", cacheKey))
			lc.Acquire()
			defer lc.Release()

			if value, ok := handleGetCache(client, cacheKey); ok {
				return value
			}
		}

		// 缓存没有，进行实时请求
		req := handleRequest(arg)

		var respData any
		if *arg.OnlyData {
			respData = req.ResponseBody()
		} else {
			respData = map[string]any{
				"data":    req.ResponseBody(),
				"status":  req.ResponseCode(),
				"header":  req.ResponseHeader(),
				"cookies": req.ResponseCookies(),
			}
		}

		if arg.IsCache {
			// 进行数据缓存
			str, _ := json.MarshalToString(respData)
			client.Set(context.TODO(), cacheKey, str, 5*time.Minute)
		}

		// 返回数据
		value, _ := r.vm.ToValue(respData)
		return value
	}

}

// LogModule 设置log包
func LogModule(r *runtime) map[string]func(call otto.FunctionCall) otto.Value {
	transArgs := func(list []otto.Value) []any {
		var args []any
		for _, item := range list {
			if value, err := item.Export(); err != nil {
				args = append(args, "不支持的数据格式")
			} else {
				args = append(args, value)
			}
		}
		return args
	}

	return map[string]func(call otto.FunctionCall) otto.Value{
		"info": func(call otto.FunctionCall) otto.Value {
			r.ctx.Log.Info("script log", zap.Any("args", transArgs(call.ArgumentList)))
			return otto.Value{}
		},
		"warn": func(call otto.FunctionCall) otto.Value {
			r.ctx.Log.Warn("script log", zap.Any("args", transArgs(call.ArgumentList)))
			return otto.Value{}
		},
		"error": func(call otto.FunctionCall) otto.Value {
			r.ctx.Log.Error("script log", zap.Any("args", transArgs(call.ArgumentList)))
			return otto.Value{}
		},
		"debug": func(call otto.FunctionCall) otto.Value {
			r.ctx.Log.Debug("script log", zap.Any("args", transArgs(call.ArgumentList)))
			return otto.Value{}
		},
	}
}

// StoreModule 设置全局存储器
func StoreModule(r *runtime) map[string]func(call otto.FunctionCall) otto.Value {
	const storePrefixKey = "global_store"

	return map[string]func(call otto.FunctionCall) otto.Value{
		"load": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) == 0 {
				return otto.Value{}
			}
			key, err := call.Argument(0).Export()
			if err != nil {
				return otto.Value{}
			}
			r.runStore.GetData(fmt.Sprintf("%v.%v", storePrefixKey, key))
			return otto.Value{}
		},
		"store": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) < 2 {
				return otto.Value{}
			}
			key, err := call.Argument(0).Export()
			if err != nil {
				return otto.Value{}
			}

			value, err := call.Argument(0).Export()
			if err != nil {
				return otto.Value{}
			}

			r.runStore.SetData(fmt.Sprintf("%v.%v", storePrefixKey, key), value)
			return otto.Value{}
		},
	}
}

// LogIDModule 获取链路日志
func LogIDModule(r *runtime) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		if value, err := call.Otto.ToValue(r.ctx.TraceID); err != nil {
			return otto.Value{}
		} else {
			return value
		}
	}
}

// TrxModule 获取请求唯一id
func TrxModule(r *runtime) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		if value, err := call.Otto.ToValue(r.trx); err != nil {
			return otto.Value{}
		} else {
			return value
		}
	}
}

// ActiveBreakModule 主动中断请求
func ActiveBreakModule(r *runtime) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		if len(call.ArgumentList) == 0 {
			panic(NewModuleArgError("break method argument not null"))
		}
		if !call.Argument(0).IsString() {
			panic(NewModuleArgError("break method argument must is string"))
		}
		panic(NewActiveBreakError(call.Argument(0).String()))
		return otto.Value{}
	}
}

// ActiveSuspendModule 主动挂起请求
func ActiveSuspendModule(r *runtime) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		if len(call.ArgumentList) == 0 {
			panic(NewModuleArgError("suspend method argument not null"))
		}

		// 1:msg
		if len(call.ArgumentList) == 1 {
			if !call.Argument(0).IsString() {
				panic(NewModuleArgError("suspend method argument must is string"))
			}
			// 挂起
			panic(NewActiveSuspendError("", call.Argument(1).String()))
		}
		// 1:code 2:msg
		if len(call.ArgumentList) >= 2 {
			if !call.Argument(0).IsString() || !call.Argument(1).IsString() {
				panic(NewModuleArgError("suspend method argument must is string"))
			}
			// 挂起
			panic(NewActiveSuspendError(call.Argument(0).String(), call.Argument(1).String()))
		}
		return otto.Value{}
	}
}
