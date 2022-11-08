package engine

import (
	"fmt"
	"github.com/limeschool/gin"
	"github.com/robertkrimen/otto"
	"go.uber.org/zap"
	"ps-go/tools"
)

// GetGlobalJsModule 全局module 函数
func GetGlobalJsModule(r *runtime) any {
	return gin.H{
		"request": getHttpModule(r),
		"log":     getLogModule(r),
		"data":    getStoreModule(r),
	}
}

// todo 这里需要加上链路日志
// getHttpModule 设置http 请求函数
func getHttpModule(r *runtime) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		if len(call.ArgumentList) == 0 {
			panic("request method argument not null")
		}

		param, err := call.Argument(0).Export()
		if err != nil {
			panic(fmt.Sprint("request method argument type err:", err.Error()))
		}

		byteData, err := json.Marshal(param)

		request := tools.HttpRequest{}
		if err = json.Unmarshal(byteData, &request); err != nil {
			panic(fmt.Sprint("request method argument field err:", err.Error()))
		}

		response, err := request.Do()
		if err != nil {
			panic(fmt.Sprint("send http request err:", err.Error()))
		}

		value, _ := otto.ToValue(response)
		return value
	}
}

// getLogModule 设置log包
func getLogModule(r *runtime) map[string]func(call otto.FunctionCall) otto.Value {
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

// getStoreModule 设置全局存储器
func getStoreModule(r *runtime) map[string]func(call otto.FunctionCall) otto.Value {
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

// getContextLogID 获取链路日志
func getContextLogID(r *runtime) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		if value, err := call.Otto.ToValue(r.ctx.TraceID); err != nil {
			return otto.Value{}
		} else {
			return value
		}
	}
}
