package tools

import (
	"encoding/xml"
	"fmt"
	"github.com/valyala/fasthttp"
	"ps-go/consts"
	"ps-go/errors"
	"strings"
)

type HttpRequest struct {
	Url         string
	Method      string
	Body        any
	Header      map[string]string
	Auth        []string
	ContentType string
	Timeout     int
	RespType    string
}

func (r *HttpRequest) Do() (any, error) {
	if r.Url == "" {
		return nil, errors.New("request Url not empty")
	}

	if r.Method == "" {
		return nil, errors.New("request Method not empty")
	}

	r.Method = strings.ToUpper(r.Method)

	// 默认60秒
	if r.Timeout == 0 {
		r.Timeout = 60
	}

	// 默认请求类型为json
	if r.ContentType == "" {
		r.ContentType = "application/json"
	}

	if r.RespType == "" {
		r.RespType = consts.RespJson
	}

	// 处理请求body
	var data []byte
	if r.Body != nil {
		if r.Method == "GET" {
			r.Url += "?" + r.BodyToQuery(r.Body)
		} else {
			data, _ = json.Marshal(r.Body)
		}
	}

	// 用完需要释放资源
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	// 设置请求信息
	if len(r.Auth) == 2 {
		req.Header.Set("Basic Auth", fmt.Sprintf("%v %v", r.Auth[0], r.Auth[1]))
	}

	// 设置请求头
	if len(r.Header) != 0 {
		for key, val := range r.Header {
			req.Header.Set(key, val)
		}
	}

	req.SetRequestURI(r.Url)
	req.Header.SetContentType(r.ContentType)
	req.Header.SetMethod(r.Method)
	req.SetBody(data)

	// 用完需要释放资源
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// 发起请求
	if err := fasthttp.Do(req, resp); err != nil {
		return nil, err
	}

	// 处理返回值
	b := resp.Body()

	if r.RespType == consts.RespJson {
		var respData = make(map[string]any)
		if json.Unmarshal(b, &respData) != nil {
			return nil, errors.New("返回数据非json格式")
		}
		return respData, nil
	}

	if r.RespType == consts.RespXml {
		var respData = make(map[string]any)
		if xml.Unmarshal(b, (*XmlResult)(&respData)) != nil {
			return nil, errors.New("返回数据非xml格式")
		}
		return respData, nil
	}

	if r.RespType == consts.RespText {
		return string(b), nil
	}

	return nil, errors.NewF("非法的数据返回格式:%v", r.RespType)
}

func (r *HttpRequest) BodyToQuery(data any) string {
	var slice []string
	switch data.(type) {
	case map[string]any:
		for key, val := range data.(map[string]any) {
			slice = append(slice, fmt.Sprintf("%v=%v", key, val))
		}
	case map[string]string:
		for key, val := range data.(map[string]any) {
			slice = append(slice, fmt.Sprintf("%v=%v", key, val))
		}
	case map[string]int:
		for key, val := range data.(map[string]any) {
			slice = append(slice, fmt.Sprintf("%v=%v", key, val))
		}
	case map[string]float64:
		for key, val := range data.(map[string]any) {
			slice = append(slice, fmt.Sprintf("%v=%v", key, val))
		}
	}
	return strings.Join(slice, "&")
}
