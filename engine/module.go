package engine

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"github.com/go-redis/redis/v8"
	json "github.com/json-iterator/go"
	"github.com/limeschool/gin"
	"github.com/robertkrimen/otto"
	"go.uber.org/zap"
	"ps-go/consts"
	"ps-go/model"
	"ps-go/tools"
	"ps-go/tools/aes"
	"ps-go/tools/lock"
	"ps-go/tools/rsa"
	"time"
	"unsafe"
)

// GetGlobalJsModule 全局module 函数
func GetGlobalJsModule(r *runtime) any {
	return gin.H{
		"request":  RequestModule(r),      //发送请求
		"log":      LogModule(r),          //打印日志
		"data":     StoreModule(r),        //存储数据
		"logId":    LogIDModule(r),        //获取logId
		"trx":      TrxModule(r),          //获取trx
		"suspend":  ActiveSuspendModule(), //主动挂起
		"break":    ActiveBreakModule(),   //主动中断
		"response": ResponseModule(r),     //主动返回
		"base64":   Base64Module(),        //base64加解密
		"uuid":     UuidModule(),          //生成唯一id
		"aes":      AesModule(),           //aes加解密
		"rsa":      RsaModule(r),          //rsa加解密
	}
}

// RequestModule 设置http 请求函数，返回详细请求信息包括header头
func RequestModule(r *runtime) func(call otto.FunctionCall) otto.Value {

	type tls struct {
		Ca  string `json:"ca"`
		Key string `json:"key"`
	}

	type requestArg struct {
		Url          string            `json:"url"`          //请求的url
		Method       string            `json:"method"`       //请求的方法
		Body         any               `json:"body"`         //请求的body
		Header       map[string]string `json:"header"`       //请求的header
		Auth         []string          `json:"auth"`         //请求的auth
		ContentType  string            `json:"contentType"`  //请求类型
		RequestType  string            `json:"requestType"`  //数据类型
		Timeout      int               `json:"timeout"`      //超时时间
		ResponseType string            `json:"responseType"` //返回类型
		IsCache      bool              `json:"isCache"`      //是否缓存
		OnlyData     *bool             `json:"onlyData"`     //是否只返回data,不携带header等 默认true
		Tls          *tls              `json:"tls"`          //请求需要携带证书时使用
	}

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
			RequestType:  arg.RequestType,
			Timeout:      arg.Timeout,
			ResponseType: arg.ResponseType,
		}

		if arg.Tls != nil {
			caSecret := model.Secret{}
			if err = caSecret.OneByName(r.ctx, arg.Tls.Ca); err != nil {
				panic(NewModuleArgError(fmt.Sprintf("request method tls.ca name found err :%v", err.Error())))
			}
			keySecret := model.Secret{}
			if err = keySecret.OneByName(r.ctx, arg.Tls.Key); err != nil {
				panic(NewModuleArgError(fmt.Sprintf("request method tls.key name found err :%v", err.Error())))
			}

			request.Tls = &tools.Tls{
				Ca:  []byte(caSecret.Context),
				Key: []byte(keySecret.Context),
			}
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
func ActiveBreakModule() func(call otto.FunctionCall) otto.Value {
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
func ActiveSuspendModule() func(call otto.FunctionCall) otto.Value {
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

// ResponseModule 主动返回请求
func ResponseModule(r *runtime) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		if len(call.ArgumentList) == 0 {
			panic(NewModuleArgError("response method argument not null"))
		}

		if !call.Argument(0).IsObject() {
			panic(NewModuleArgError("response method argument must is object"))
		}

		respData := map[string]any{}
		byteData, _ := call.Argument(0).MarshalJSON()
		_ = json.Unmarshal(byteData, &respData)

		// 将返回的值设置到存储器中
		r.response.SetAndClose(respData)

		return otto.Value{}
	}
}

// Base64Module base64加解密
func Base64Module() map[string]func(call otto.FunctionCall) otto.Value {
	return map[string]func(call otto.FunctionCall) otto.Value{
		"encode": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) == 0 {
				panic(NewModuleArgError("base64.encode method argument not null"))
			}

			if !call.Argument(0).IsString() {
				panic(NewModuleArgError("base64.encode method argument must is string"))
			}

			str := call.Argument(0).String()
			enStr := base64.StdEncoding.EncodeToString(*(*[]byte)(unsafe.Pointer(&str)))
			value, _ := call.Otto.ToValue(enStr)
			return value
		},
		"decode": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) == 0 {
				panic(NewModuleArgError("base64.decode method argument is string"))
			}
			if !call.Argument(0).IsString() {
				panic(NewModuleArgError("base64.decode method argument must is string"))
			}

			str := call.Argument(0).String()
			enStr := base64.StdEncoding.EncodeToString(*(*[]byte)(unsafe.Pointer(&str)))
			value, _ := call.Otto.ToValue(enStr)
			return value
		},
	}
}

// UuidModule 生成唯一id
func UuidModule() func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		value, _ := call.Otto.ToValue(tools.UUID())
		return value
	}
}

// AesModule aes加解密
func AesModule() map[string]func(call otto.FunctionCall) otto.Value {
	return map[string]func(call otto.FunctionCall) otto.Value{
		"encodeToBase64": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) != 2 {
				panic(NewModuleArgError("aes.encodeToBase64 method has only two parameters"))
			}

			if !call.Argument(0).IsString() || !call.Argument(1).IsString() {
				panic(NewModuleArgError("aes.encodeToBase64 method parameter must be string"))
			}

			enStr, err := aes.EncryptToBase64(call.Argument(1).String(), call.Argument(0).String())
			if err != nil {
				panic(NewModuleArgError(fmt.Sprintf("aes.encodeToBase64 err:%v", err)))
			}

			value, _ := call.Otto.ToValue(enStr)
			return value
		},
		"decodeFromBase64": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) != 2 {
				panic(NewModuleArgError("aes.decodeFromBase64 method has only two parameters"))
			}

			if !call.Argument(0).IsString() || !call.Argument(1).IsString() {
				panic(NewModuleArgError("aes.decodeFromBase64 method parameter must be string"))
			}

			enStr, err := aes.DecryptFromBase64(call.Argument(1).String(), call.Argument(0).String())
			if err != nil {
				panic(NewModuleArgError(fmt.Sprintf("aes.decodeFromBase64 err:%v", err)))
			}

			value, _ := call.Otto.ToValue(enStr)
			return value
		},
		"encodeToHex": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) != 2 {
				panic(NewModuleArgError("aes.encodeToHex method has only two parameters"))
			}

			if !call.Argument(0).IsString() || !call.Argument(1).IsString() {
				panic(NewModuleArgError("aes.encodeToHex method parameter must be string"))
			}

			enStr, err := aes.EncryptToHex(call.Argument(1).String(), call.Argument(0).String())
			if err != nil {
				panic(NewModuleArgError(fmt.Sprintf("aes.encodeToHex err:%v", err)))
			}

			value, _ := call.Otto.ToValue(enStr)
			return value
		},
		"decodeFromHex": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) != 2 {
				panic(NewModuleArgError("aes.decodeFromHex method has only two parameters"))
			}

			if !call.Argument(0).IsString() || !call.Argument(1).IsString() {
				panic(NewModuleArgError("aes.decodeFromHex method parameter must be string"))
			}

			enStr, err := aes.DecryptFromHex(call.Argument(1).String(), call.Argument(0).String())
			if err != nil {
				panic(NewModuleArgError(fmt.Sprintf("aes.decodeFromHex err:%v", err)))
			}

			value, _ := call.Otto.ToValue(enStr)
			return value
		},
	}
}

// RsaModule rsa加解密
func RsaModule(r *runtime) map[string]func(call otto.FunctionCall) otto.Value {
	findKey := func(k string) []byte {
		secret := model.Secret{}
		if err := secret.OneByName(r.ctx, k); err != nil {
			panic(NewModuleArgError(fmt.Sprintf("rsa secret name %v does not exist", k)))
		}
		return []byte(secret.Context)
	}

	return map[string]func(call otto.FunctionCall) otto.Value{
		"encodeToBase64": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) != 2 {
				panic(NewModuleArgError("rsa.encodeToBase64 method has only two parameters"))
			}

			if !call.Argument(0).IsString() || !call.Argument(1).IsString() {
				panic(NewModuleArgError("rsa.encodeToBase64 method parameter must be string"))
			}

			enStr, err := rsa.EncryptToBase64(call.Argument(1).String(), findKey(call.Argument(0).String()))
			if err != nil {
				panic(NewModuleArgError(fmt.Sprintf("rsa.encodeToBase64 err:%v", err)))
			}

			value, _ := call.Otto.ToValue(enStr)
			return value
		},
		"decodeFromBase64": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) != 2 {
				panic(NewModuleArgError("rsa.decodeFromBase64 method has only two parameters"))
			}

			if !call.Argument(0).IsString() || !call.Argument(1).IsString() {
				panic(NewModuleArgError("rsa.decodeFromBase64 method parameter must be string"))
			}

			enStr, err := rsa.DecryptFromBase64(call.Argument(1).String(), findKey(call.Argument(0).String()))
			if err != nil {
				panic(NewModuleArgError(fmt.Sprintf("rsa.decodeFromBase64 err:%v", err)))
			}

			value, _ := call.Otto.ToValue(enStr)
			return value
		},
		"encodeToHex": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) != 2 {
				panic(NewModuleArgError("rsa.encodeToHex method has only two parameters"))
			}

			if !call.Argument(0).IsString() || !call.Argument(1).IsString() {
				panic(NewModuleArgError("rsa.encodeToHex method parameter must be string"))
			}

			enStr, err := rsa.EncryptToHex(call.Argument(1).String(), findKey(call.Argument(0).String()))
			if err != nil {
				panic(NewModuleArgError(fmt.Sprintf("rsa.encodeToHex err:%v", err)))
			}

			value, _ := call.Otto.ToValue(enStr)
			return value
		},
		"decodeFromHex": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) != 2 {
				panic(NewModuleArgError("rsa.decodeFromHex method has only two parameters"))
			}

			if !call.Argument(0).IsString() || !call.Argument(1).IsString() {
				panic(NewModuleArgError("rsa.decodeFromHex method parameter must be string"))
			}

			enStr, err := rsa.DecryptFromHex(call.Argument(1).String(), findKey(call.Argument(0).String()))
			if err != nil {
				panic(NewModuleArgError(fmt.Sprintf("rsa.decodeFromHex err:%v", err)))
			}

			value, _ := call.Otto.ToValue(enStr)
			return value
		},
	}
}
