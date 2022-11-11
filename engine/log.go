package engine

import (
	"fmt"
	"ps-go/tools"
	"sync"
	"time"
)

type Logger interface {
	SetResponseTime()
	SetRunTime()
	SetRequest(data any)
	SetResponse(data any)
	SetStep(step int)
	SetError(err error)
	SetStatus(status string)
	NewStepLog(step, ac int) StepLog
	GetString() string
	SetStartTime(time.Time)
	GetStepErr(int) StepLog
}

type StepLog interface {
	SetRunTime(t time.Time)
	SetError(err error)
	NewComponentLog(step, action int) ComponentLog
}

type ComponentLog interface {
	SetResponse(resp any)
	SetRequest(com Component)
	SetApiRequest(com tools.HttpRequest)
	SetRetryCount(c int)
	SetError(err error)
	SetRunTime(t time.Time)
	NewRequestLog() RequestLog
	SetStep(step int)
	SetAction(c int)
	SetSkip(is bool)
}

type RequestLog interface {
	SetRunTime(t time.Time)
	SetRequest(com tools.HttpRequest)
	SetRespHeader(data map[string]string)
	SetRespCode(code int)
	SetRespBody(body any)
	SetRespCookies(data map[string]string)
	SetError(err error)
}

// 日志引擎
type runLog struct {
	start time.Time
	lock  sync.RWMutex

	LogId string `json:"log_id"` //链路id
	Step  int    `json:"step"`   //总步数

	Request  any `json:"request"`  //请求参数
	Response any `json:"response"` //输出参数

	StepLogs []*stepLog `json:"step_logs"` //层级日志

	Error   string `json:"error,omitempty"` //错误原因
	Status  string `json:"status"`          //运行状态 中断/错误挂起/主动挂起/成功
	CurStep int    `json:"cur_step"`        //当前执行步数

	RespTime      string `json:"resp_time"`     //请求返回时长
	RespDatetime  string `json:"resp_datetime"` //返回的具体时间
	RunTime       string `json:"run_time"`      //调用链运行时间
	StartDatetime string `json:"start_datetime"`
	EndDatetime   string `json:"end_datetime"`
}

type stepLog struct {
	lock sync.RWMutex

	Step          int             `json:"step"`           //当前行数
	ActionCount   int             `json:"action_count"`   //需要执行的组件数量
	StartDatetime string          `json:"start_datetime"` //开始时间
	EndDatetime   string          `json:"end_datetime"`   //结束时间
	RunTime       string          `json:"run_time"`       //运行时间
	ComponentLogs []*componentLog `json:"component_logs"` //组件日志
	Error         string          `json:"error,omitempty"`
}

func (s *stepLog) SetError(err error) {
	if err != nil {
		s.Error = err.Error()
	}
}

func (s *stepLog) SetRunTime(t time.Time) {
	cur := time.Now()
	s.StartDatetime = t.Format(LogDatetimeFormat)
	s.EndDatetime = cur.Format(LogDatetimeFormat)
	s.RunTime = fmt.Sprintf("%vs", float64(time.Now().UnixMilli()-t.UnixMilli())/1000)
}

func (s *stepLog) NewComponentLog(step, action int) ComponentLog {
	s.lock.Lock()
	defer s.lock.Unlock()
	log := componentLog{
		lock:   sync.RWMutex{},
		Step:   step,
		Action: action,
	}
	s.ComponentLogs = append(s.ComponentLogs, &log)
	return &log
}

type componentLog struct {
	lock sync.RWMutex

	Step       int    `json:"step"`            //当前行数
	Action     int    `json:"action"`          //当前列数
	RetryCount int    `json:"retry_count"`     //重试次数
	Error      string `json:"error,omitempty"` //错误原因

	RunTime       string `json:"run_time"` //运行时间
	StartDatetime string `json:"start_datetime"`
	EndDatetime   string `json:"end_datetime"`

	Input      map[string]any `json:"input"`       //输入数据
	Name       string         `json:"name"`        //组件名
	Desc       string         `json:"desc"`        //组件描述
	Type       string         `json:"type"`        //组件类型 [api|script]
	Url        string         `json:"url"`         //地址
	OutputName string         `json:"output_name"` //输出对象名
	IsCache    bool           `json:"is_cache"`    //是否启用缓存
	IsSkip     bool           `json:"is_skip"`     //是否进入执行
	// api 特有日志字段
	Method       string            `json:"method,omitempty"`
	Body         any               `json:"body,omitempty"`
	Header       map[string]string `json:"header,omitempty"`
	Auth         []string          `json:"auth,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	DataType     string            `json:"data_type,omitempty"` //xml|text|json
	Timeout      int               `json:"timeout,omitempty"`
	ResponseType string            `json:"response_type,omitempty"`

	Response    any           `json:"response"`               //输出数据
	RequestLogs []*requestLog `json:"request_logs,omitempty"` //使用脚本请求的数据
}

func (s *componentLog) SetStep(step int) {
	s.Step = step
}

func (s *componentLog) SetSkip(is bool) {
	s.IsSkip = is
}

func (s *componentLog) SetAction(c int) {
	s.Action = c
}

func (s *componentLog) NewRequestLog() RequestLog {
	s.lock.Lock()
	defer s.lock.Unlock()
	log := requestLog{}
	s.RequestLogs = append(s.RequestLogs, &log)
	return &log
}

func (s *componentLog) SetResponse(resp any) {
	s.Response = resp
}

func (s *componentLog) SetRequest(com Component) {
	s.Input = com.Input
	s.Name = com.Name
	s.Desc = com.Desc
	s.Type = com.Type
	s.Url = com.Url
	s.OutputName = com.OutputName
	s.IsCache = com.IsCache
}

func (s *componentLog) SetApiRequest(com tools.HttpRequest) {
	s.Method = com.Method
	s.Body = com.Body
	s.Header = com.Header
	s.Auth = com.Auth
	s.ContentType = com.ContentType
	s.DataType = com.DataType
	s.Timeout = com.Timeout
	s.ResponseType = com.ResponseType
}

func (s *componentLog) SetRetryCount(c int) {
	s.RetryCount = c
}

func (s *componentLog) SetError(err error) {
	if err != nil {
		s.Error = err.Error()
	}
}

func (s *componentLog) SetRunTime(t time.Time) {
	cur := time.Now()
	s.StartDatetime = t.Format(LogDatetimeFormat)
	s.EndDatetime = cur.Format(LogDatetimeFormat)
	s.RunTime = fmt.Sprintf("%vs", float64(time.Now().UnixMilli()-t.UnixMilli())/1000)
}

type requestLog struct {
	Url           string            `json:"url,omitempty"`
	Method        string            `json:"method,omitempty"`
	Body          any               `json:"body,omitempty"`
	Header        map[string]string `json:"header,omitempty"`
	Auth          []string          `json:"auth,omitempty"`
	ContentType   string            `json:"content_type,omitempty"`
	DataType      string            `json:"data_type,omitempty"`
	Timeout       int               `json:"timeout,omitempty"`
	ResponseType  string            `json:"response_type,omitempty"`
	RespHeader    map[string]string `json:"resp_header,omitempty"`
	RespCode      int               `json:"resp_code,omitempty"`
	RespBody      any               `json:"resp_body,omitempty"`
	RespCookies   map[string]string `json:"resp_cookies,omitempty"`
	Error         string            `json:"error,omitempty"`
	RunTime       string            `json:"run_time,omitempty"`
	StartDatetime string            `json:"start_datetime,omitempty"`
	EndDatetime   string            `json:"end_datetime,omitempty"`
}

func (s *requestLog) SetRequest(com tools.HttpRequest) {
	timeout := com.Timeout
	if timeout <= 0 || timeout > 60 {
		timeout = 60
	}
	s.Url = com.Url
	s.Method = com.Method
	s.Body = com.Body
	s.Header = com.Header
	s.Auth = com.Auth
	s.ContentType = com.ContentType
	s.DataType = com.DataType
	s.Timeout = com.Timeout
	s.ResponseType = com.ResponseType
}

func (s *requestLog) SetRespHeader(data map[string]string) {
	s.RespHeader = data
}

func (s *requestLog) SetRespCode(code int) {
	s.RespCode = code
}

func (s *requestLog) SetError(err error) {
	if err != nil {
		s.Error = err.Error()
	}
}

func (s *requestLog) SetRespBody(body any) {
	s.Body = body
}

func (s *requestLog) SetRespCookies(data map[string]string) {
	s.RespCookies = data
}

func (s *requestLog) SetRunTime(t time.Time) {
	cur := time.Now()
	s.StartDatetime = t.Format(LogDatetimeFormat)
	s.EndDatetime = cur.Format(LogDatetimeFormat)
	s.RunTime = fmt.Sprintf("%vs", float64(time.Now().UnixMilli()-t.UnixMilli())/1000)
}

func (r *runLog) NewStepLog(step, ac int) StepLog {
	r.lock.Lock()
	defer r.lock.Unlock()
	log := stepLog{
		lock:        sync.RWMutex{},
		Step:        step,
		ActionCount: ac,
	}
	r.StepLogs = append(r.StepLogs, &log)
	return &log
}

func (r *runLog) SetRequest(data any) {
	r.Request = data
}

func (r *runLog) SetStep(step int) {
	r.CurStep = step
}

func (r *runLog) SetError(err error) {
	if err != nil {
		r.Error = err.Error()
	}
}

func (r *runLog) SetStatus(status string) {
	r.Status = status
}

func (r *runLog) SetResponse(data any) {
	r.Response = data
}

func (r *runLog) SetResponseTime() {
	cur := time.Now()
	r.RespDatetime = cur.Format(LogDatetimeFormat)
	r.RespTime = fmt.Sprintf("%vs", float64(time.Now().UnixMilli()-r.start.UnixMilli())/1000)
}

// SetRunTime 计算开始时间和结束时间以及使用时间
func (r *runLog) SetRunTime() {
	cur := time.Now()
	r.StartDatetime = r.start.Format(LogDatetimeFormat)
	r.EndDatetime = cur.Format(LogDatetimeFormat)
	r.RunTime = fmt.Sprintf("%vs", float64(time.Now().UnixMilli()-r.start.UnixMilli())/1000)
}

func (r *runLog) GetString() string {
	str, _ := json.MarshalToString(r)
	return str
}

func (r *runLog) SetStartTime(t time.Time) {
	r.start = t
}

func (r *runLog) GetStepErr(index int) StepLog {
	if index > len(r.StepLogs) {
		return nil
	}
	return r.StepLogs[index]
}
