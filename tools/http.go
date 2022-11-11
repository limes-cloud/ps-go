package tools

import (
	"encoding/xml"
	"fmt"
	"github.com/valyala/fasthttp"
	"ps-go/consts"
	"ps-go/errors"
	"strings"
	"unsafe"
)

type HttpRequest struct {
	Url          string            `json:"url"`
	Method       string            `json:"method"`
	Body         any               `json:"body"`
	Header       map[string]string `json:"header"`
	Auth         []string          `json:"auth"`
	ContentType  string            `json:"content_type"`
	DataType     string            `json:"data_type"` //xml|text|json
	Timeout      int               `json:"timeout"`
	ResponseType string            `json:"response_type"`

	// 返回数据
	respHeader  map[string]string `json:"-"`
	respCode    int               `json:"-"`
	respBody    any               `json:"-"`
	respCookies map[string]string `json:"-"`
}

func (r *HttpRequest) ResponseHeader() map[string]string {
	return r.respHeader
}

func (r *HttpRequest) ResponseCode() int {
	return r.respCode
}

func (r *HttpRequest) ResponseBody() any {
	return r.respBody
}

func (r *HttpRequest) ResponseCookies() map[string]string {
	return r.respCookies
}

func (r *HttpRequest) Result() (any, error) {
	err := r.Do()
	return r.Body, err
}

func (r *HttpRequest) Do() error {
	if r.Url == "" {
		return errors.New("request url not empty")
	}

	if r.Method == "" {
		return errors.New("request method not empty")
	}

	r.Method = strings.ToUpper(r.Method)

	// 默认60秒
	if r.Timeout <= 0 || r.Timeout > 60 {
		r.Timeout = 60
	}

	// 默认的数据类型为json
	if r.DataType == "" {
		r.DataType = consts.RespJson
	}

	// 默认的请求类型根据数据类型赋值
	if r.ContentType == "" && r.DataType != consts.RespText {
		r.ContentType = "application/" + r.DataType
	}

	// 处理请求body
	var data []byte
	if r.Body != nil {
		if r.Method == "GET" {
			r.Url += "?" + r.BodyToQuery()
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

	if r.ContentType != "" {
		req.Header.SetContentType(r.ContentType)
	}

	req.SetRequestURI(r.Url)
	req.Header.SetMethod(r.Method)
	req.SetBody(data)

	// 用完需要释放资源
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// 发起请求
	if err := fasthttp.Do(req, resp); err != nil {
		return err
	}

	// 获取返回信息

	r.respCode = resp.StatusCode()
	r.respHeader = r.getHeader(resp)
	r.respCookies = r.getCookies(r.respHeader)
	// 处理返回值
	b := resp.Body()

	if r.ResponseType == consts.RespJson {
		var respData = make(map[string]any)
		if json.Unmarshal(b, &respData) != nil {
			return errors.New("返回数据非json格式")
		}
		r.respBody = respData
		return nil
	}

	if r.ResponseType == consts.RespXml {
		var respData = make(map[string]any)
		if xml.Unmarshal(b, (*XmlResult)(&respData)) != nil {
			return errors.New("返回数据非xml格式")
		}
		r.respBody = respData
		return nil
	}

	if r.ResponseType == consts.RespText {
		r.respBody = (*string)(unsafe.Pointer(&b))
		return nil
	}

	return errors.NewF("非法的数据返回格式:%v", r.ResponseType)
}

func (r *HttpRequest) BodyToQuery() string {
	data := r.Body
	if r.DataType == consts.RespXml {
		respData := make(map[string]any)
		if xml.Unmarshal([]byte(fmt.Sprint(r.Body)), (*XmlResult)(&respData)) == nil {
			data = respData
		}
	}

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

func (r *HttpRequest) getCookies(header map[string]string) map[string]string {
	respData := make(map[string]string)

	cookie := header["Set-Cookie"]
	if cookie == "" {
		cookie = header["Cookie"]
	}

	if cookie == "" {
		return respData
	}

	cookie = strings.ReplaceAll(cookie, " ", "")
	cs := strings.Split(cookie, ";")
	for _, val := range cs {
		ck := strings.Split(val, "=")
		if len(ck) == 2 {
			respData[ck[0]] = ck[1]
		}
	}
	return respData
}

func (r *HttpRequest) getHeader(resp *fasthttp.Response) map[string]string {
	str := resp.Header.String()
	slice := strings.Split(str, "\r\n")

	header := map[string]string{}

	for _, lineStr := range slice {
		index := strings.Index(lineStr, ":")
		if index == -1 {
			continue
		}
		key := strings.Trim(lineStr[:index], " ")
		val := strings.Trim(lineStr[index+1:], " ")
		header[key] = val
	}

	return header
}
