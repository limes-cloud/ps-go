package engine

import (
	"encoding/xml"
	"fmt"
	"github.com/valyala/fasthttp"
	"ps-go/consts"
	"ps-go/errors"
	"ps-go/tools"
	"strings"
)

type HttpRequest struct {
	url         string
	method      string
	body        any
	header      map[string]string
	auth        []string
	contentType string
	timeout     int
	respType    string
}

func (r *HttpRequest) Do() (any, error) {
	if r.url == "" {
		return nil, errors.New("request url not empty")
	}

	if r.method == "" {
		return nil, errors.New("request method not empty")
	}

	r.method = strings.ToUpper(r.method)

	// 默认60秒
	if r.timeout == 0 {
		r.timeout = 60
	}

	// 默认请求类型为json
	if r.contentType == "" {
		r.contentType = "application/json"
	}

	if r.respType == "" {
		r.respType = consts.RespJson
	}

	// 处理请求body
	var data []byte
	if r.body != nil {
		data, _ = json.Marshal(r.body)
	}

	// 用完需要释放资源
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	// 设置请求信息
	if len(r.auth) == 2 {
		req.Header.Set("Basic Auth", fmt.Sprintf("%v %v", r.auth[0], r.auth[1]))
	}

	// 设置请求头
	if len(r.header) != 0 {
		for key, val := range r.header {
			req.Header.Set(key, val)
		}
	}

	req.SetRequestURI(r.url)
	req.Header.SetContentType(r.contentType)
	req.Header.SetMethod(r.method)
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

	if r.respType == consts.RespJson {
		var respData = make(map[string]any)
		if json.Unmarshal(b, &respData) != nil {
			return nil, errors.New("返回数据非json格式")
		}
		return respData, nil
	}

	if r.respType == consts.RespXml {
		var respData = make(map[string]any)
		if xml.Unmarshal(b, (*tools.XmlResult)(&respData)) != nil {
			return nil, errors.New("返回数据非xml格式")
		}
		return respData, nil
	}

	if r.respType == consts.RespText {
		return string(b), nil
	}

	return nil, errors.NewF("非法的数据返回格式:%v", r.respType)
}
