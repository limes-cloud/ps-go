package engine

import (
	"errors"
	"fmt"
	"ps-go/tools"
)

type Rule struct {
	Version    string        `json:"version"`
	Record     bool          `json:"record"`     //是否记录流程数据
	Suspend    bool          `json:"suspend"`    //是否开启异常中断挂起 [脚本错误/异常捕捉错误]
	Request    Request       `json:"request"`    //请求信息
	Response   Response      `json:"response"`   //返回信息
	Components [][]Component `json:"components"` //组件信息
}

type Request struct {
	Type   string               `json:"type"`             //body数据类型
	Query  map[string]FieldRule `json:"query,omitempty"`  //query参数
	Body   map[string]FieldRule `json:"body,omitempty"`   //body参数
	Header map[string]FieldRule `json:"header,omitempty"` //请求头
}

type FieldRule struct {
	Type      string               `json:"type"`                //字段类型 [any]
	Attribute map[string]FieldRule `json:"attribute,omitempty"` //字段属性 [object]
	Required  bool                 `json:"required"`            //是否必填 [any]
	Default   any                  `json:"default,omitempty"`   //字段默认值 [any]
	MaxLen    *int                 `json:"maxLen,omitempty"`    //最大长度 [string]
	MinLen    *int                 `json:"minLen,omitempty"`    //最小长度 [string]
	Max       any                  `json:"max,omitempty"`       //最大值 [integer|float]
	Min       any                  `json:"min,omitempty"`       //最小值 [integer|float]
	Enum      []any                `json:"enum,omitempty"`      //枚举值 [integer|float|string|slice]
}

type Response struct {
	Type        string         `json:"type"`        //返回数据类型，目前仅支持json
	Body        map[string]any `json:"body"`        //返回数据
	Header      map[string]any `json:"header"`      //返回附加header
	DefaultBody map[string]any `json:"defaultBody"` //默认返回值
}

type tls struct {
	Ca  string `json:"ca"`
	Key string `json:"key"`
}

type Component struct {
	IsFinish  bool           `json:"-"`                   //附加字段，恢复任务时用
	Name      string         `json:"name"`                //组件名,同一个step层下，name不能重复
	Desc      string         `json:"desc"`                //组件描述
	Type      string         `json:"type"`                //组件类型 [api|script]
	Input     map[string]any `json:"input,omitempty"`     //输入参数
	Condition string         `json:"condition,omitempty"` //准入条件
	Url       string         `json:"url"`                 //组件地址|api接口
	IsCache   bool           `json:"isCache"`             //是否启用缓存

	Method            string         `json:"method,omitempty"`       //请求方法，仅api支持
	ContentType       string         `json:"contentType,omitempty"`  //数据类型，仅api支持
	Auth              []any          `json:"auth,omitempty"`         //请求auth，仅api支持
	Header            map[string]any `json:"header,omitempty"`       //请求header，仅api支持
	ResponseType      string         `json:"responseType,omitempty"` //返回数据类型，仅api支持
	DataType          string         `json:"dataType,omitempty"`     //数据类型，仅api支持
	Tls               *tls           `json:"tls,omitempty"`          //请求ca证书，仅api支持
	ResponseCondition string         `json:"responseCondition"`      //返回判断条件，仅api支持
	ErrorMsg          string         `json:"errorMsg"`               //返回不符合条件时，返回的错误码，仅api支持

	NowResponse   bool   `json:"nowResponse"`   //是否立即响应
	IgnoreError   bool   `json:"ignoreError"`   //是否忽略error
	OutputData    any    `json:"outputData"`    //返回数据,仅支持string 和 map
	Timeout       int    `json:"timeout"`       //组件最大运行时间/s
	OutputName    string `json:"outputName"`    //返回数据名
	RetryMaxCount int    `json:"retryMaxCount"` //最大重试次数
	RetryMaxWait  int    `json:"retryMaxWait"`  //重试最大等待时长
}

func (f *FieldRule) ValidateInt(val any, is bool) (resp int, ignore bool, err error) {
	// validate required
	if !is && f.Required {
		err = errors.New("field must required")
		return
	}

	//validate default value
	if !is {
		if f.Default == nil {
			ignore = true
		}
		resp, err = tools.ToInt(f.Default)
		return
	}

	//validate input data
	if resp, err = tools.ToInt(val); err != nil {
		return
	}

	// 判断最大值
	if f.Max != nil {
		if max, er := tools.ToInt(f.Max); er == nil && max < resp {
			err = errors.New("cannot be higher than the maximum value")
			return
		}
	}

	// 判断最小值
	if f.Min != nil {
		if min, er := tools.ToInt(f.Max); er == nil && min > resp {
			err = errors.New("cannot be lower than the minimum value")
			return
		}
	}

	// 判断枚举值
	if len(f.Enum) != 0 {
		in := false
		for _, eval := range f.Enum {
			if inVal, er := tools.ToInt(eval); er == nil && inVal == resp {
				in = true
				break
			}
		}
		if !in {
			err = errors.New("not an enum value")
			return
		}
	}
	return
}

func (f *FieldRule) ValidateFloat(val any, is bool) (resp float64, ignore bool, err error) {

	// validate required
	if !is && f.Required {
		err = errors.New("field must required")
		return
	}

	//validate default value
	if !is {
		if f.Default == nil {
			ignore = true
		}
		resp, err = tools.ToFloat(f.Default)
		return
	}

	//validate input data
	if resp, err = tools.ToFloat(val); err != nil {
		return
	}

	// 判断最大值
	if f.Max != nil {
		if max, er := tools.ToFloat(f.Max); er == nil && max < resp {
			err = errors.New("cannot be higher than the maximum value")
			return
		}
	}

	// 判断最小值
	if f.Min != nil {
		if min, er := tools.ToFloat(f.Max); er == nil && min > resp {
			err = errors.New("cannot be lower than the minimum value")
			return
		}
	}

	// 判断枚举值
	if len(f.Enum) != 0 {
		in := false
		for _, eval := range f.Enum {
			if inVal, er := tools.ToFloat(eval); er == nil && inVal == resp {
				in = true
				break
			}
		}
		if !in {
			err = errors.New("not an enum value")
			return
		}
	}
	return
}

func (f *FieldRule) ValidateString(val any, is bool) (resp string, ignore bool, err error) {
	// validate required
	if !is && f.Required {
		err = errors.New("field must required")
		return
	}

	//validate default value
	if !is {
		if f.Default == nil {
			ignore = true
		}
		resp, err = tools.ToString(f.Default)
		return
	}

	//validate input data
	resp, err = tools.ToString(val)
	return
}

func (f *FieldRule) ValidateBool(val any, is bool) (resp bool, ignore bool, err error) {
	// validate required
	if !is && f.Required {
		err = errors.New("field must required")
		return
	}

	//validate default value
	if !is {
		if f.Default == nil {
			ignore = true
		}
		resp, err = tools.ToBool(f.Default)
		return
	}

	//validate input data
	resp, err = tools.ToBool(val)
	return
}

func (f *FieldRule) ValidateSlice(val any, is bool) (resp []any, ignore bool, err error) {

	// validate required
	if !is && f.Required {
		err = errors.New("field must required")
		return
	}

	//validate default value
	if !is {
		if f.Default == nil {
			ignore = true
		}
		resp, err = tools.ToSlice(f.Default)
		return
	}

	//validate input data
	if resp, err = tools.ToSlice(val); err != nil {
		return
	}

	// 判断最大值
	if f.MaxLen != nil {
		if *f.MaxLen < len(resp) {
			err = errors.New("slice length cannot be higher than the maximum value")
			return
		}
	}

	// 判断最小值
	if f.MinLen != nil {
		if *f.MaxLen > len(resp) {
			err = errors.New("slice length cannot be lower than the minimum value")
			return
		}
	}

	// 判断枚举值
	if len(f.Enum) != 0 && len(resp) != 0 {
		bucket := map[any]bool{}
		for _, eval := range f.Enum {
			bucket[eval] = true
		}

		for _, inVal := range resp {
			if _, ok := bucket[inVal]; !ok {
				err = fmt.Errorf("not an enum value")
				return
			}
		}
	}

	return
}

func (f *FieldRule) ValidateMap(val any, is bool) (resp map[string]any, ignore bool, err error) {
	//validate required
	if !is && f.Required {
		err = errors.New("field must required")
		return
	}

	//validate default value
	if !is {
		if f.Default == nil {
			ignore = true
		}
		resp, err = tools.ToMap(f.Default)
		return
	}

	//validate input data
	if resp, err = tools.ToMap(val); err != nil {
		return
	}

	// 递归遍历属性值是否正确
	var tempResp any
	var tempErr error
	for key, rule := range f.Attribute {
		temp, ok := resp[key]
		if rule.Type == Int {
			tempResp, ignore, tempErr = rule.ValidateInt(temp, ok)
		}
		if rule.Type == Float {
			tempResp, ignore, tempErr = rule.ValidateFloat(temp, ok)
		}
		if rule.Type == Bool {
			tempResp, ignore, tempErr = rule.ValidateBool(temp, ok)
		}
		if rule.Type == Slice {
			tempResp, ignore, tempErr = rule.ValidateSlice(temp, ok)
		}
		if rule.Type == Map {
			tempResp, ignore, tempErr = rule.ValidateMap(temp, ok)
		}
		if rule.Type == String {
			tempResp, ignore, tempErr = rule.ValidateString(temp, ok)
		}
		if tempErr != nil && !ignore {
			return resp, ignore, tempErr
		}
		resp[key] = tempResp
	}
	return
}

func (f *FieldRule) Validate(val any, is bool) (resp any, ignore bool, err error) {
	switch f.Type {
	case Int:
		resp, ignore, err = f.ValidateInt(val, is)
	case Float:
		resp, ignore, err = f.ValidateFloat(val, is)
	case Bool:
		resp, ignore, err = f.ValidateBool(val, is)
	case Slice:
		resp, ignore, err = f.ValidateSlice(val, is)
	case Map:
		resp, ignore, err = f.ValidateMap(val, is)
	case String:
		resp, ignore, err = f.ValidateString(val, is)
	default:
		err = fmt.Errorf("%v is wrong data type", f.Type)
	}
	return
}
