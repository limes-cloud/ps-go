package engine

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/limeschool/gin"
	"github.com/robertkrimen/otto"
	"go.uber.org/zap"
	"ps-go/consts"
	"ps-go/errors"
	"ps-go/tools"
	"time"
)

// GetGlobalJsModule 全局module 函数
func GetGlobalJsModule(r *runtime) any {
	return gin.H{
		"http":  getRequestModule(r),
		"log":   getLogModule(r),
		"data":  getStoreModule(r),
		"logId": getContextLogID(r),
	}
}

// getRequestModule 设置http 请求函数，返回详细请求信息包括header头
func getRequestModule(r *runtime) map[string]func(call otto.FunctionCall) otto.Value {

	handleArg := func(call otto.FunctionCall) []byte {
		if len(call.ArgumentList) == 0 {
			panic("request method argument not null")
		}

		param, err := call.Argument(0).Export()
		if err != nil {
			panic(fmt.Sprintf("request method argument type err:%v", err.Error()))
		}

		byteData, err := json.Marshal(param)
		if err != nil {
			panic(fmt.Sprint("request method argument must object"))
		}

		return byteData
	}

	handleRequest := func(byteData []byte) *tools.HttpRequest {

		var err error

		// 创建请求日志
		log := r.componentLog.NewRequestLog()

		request := tools.HttpRequest{}
		if err = json.Unmarshal(byteData, &request); err != nil {
			err = errors.NewF("request method argument field err:%v", err.Error())
			log.SetError(err)
			panic(err)
		}

		// 设置请求参数
		log.SetRequest(request)

		if err = request.Do(); err != nil {
			err = errors.NewF("send http request err:%v", err.Error())
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

	return map[string]func(call otto.FunctionCall) otto.Value{
		"requestAll": func(call otto.FunctionCall) otto.Value {

			argByte := handleArg(call)
			req := handleRequest(argByte)
			resp := map[string]any{
				"data":    req.ResponseBody(),
				"status":  req.ResponseCode(),
				"header":  req.ResponseHeader(),
				"cookies": req.ResponseCookies(),
			}

			value, _ := r.vm.ToValue(resp)
			return value
		},

		"requestAllCache": func(call otto.FunctionCall) otto.Value {
			argByte := handleArg(call)
			key := fmt.Sprintf("request_all_%x", md5.Sum(argByte))

			// 查询redis缓存
			client := r.ctx.Redis(consts.ProcessScheduleCache)
			if str, err := client.Get(context.TODO(), key).Result(); err == nil && str != "" {
				resp := map[string]any{}
				if json.UnmarshalFromString(str, &resp) == nil {
					value, _ := r.vm.ToValue(resp)
					return value
				}
			}

			req := handleRequest(argByte)
			body := map[string]any{
				"data":    req.ResponseBody(),
				"status":  req.ResponseCode(),
				"header":  req.ResponseHeader(),
				"cookies": req.ResponseCookies(),
			}

			// 进行数据缓存
			str, _ := json.MarshalToString(body)
			client.Set(context.TODO(), key, str, 5*time.Minute)

			// 返回数据
			value, _ := r.vm.ToValue(body)
			return value
		},

		"request": func(call otto.FunctionCall) otto.Value {
			argByte := handleArg(call)
			req := handleRequest(argByte)
			value, _ := r.vm.ToValue(req.ResponseBody())
			return value
		},

		"requestCache": func(call otto.FunctionCall) otto.Value {
			argByte := handleArg(call)
			key := fmt.Sprintf("request_%x", md5.Sum(argByte))

			// 查询redis缓存
			client := r.ctx.Redis(consts.ProcessScheduleCache)
			if str, err := client.Get(context.TODO(), key).Result(); err == nil && str != "" {
				resp := map[string]any{}
				if json.UnmarshalFromString(str, &resp) == nil {
					value, _ := r.vm.ToValue(resp)
					return value
				}
			}

			// 缓存没有，进行实时请求
			req := handleRequest(argByte)
			body := req.ResponseBody()

			// 进行数据缓存
			str, _ := json.MarshalToString(body)
			client.Set(context.TODO(), key, str, 5*time.Minute)

			// 返回数据
			value, _ := r.vm.ToValue(body)
			return value
		},
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
