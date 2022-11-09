package engine

import (
	"encoding/xml"
	"fmt"
	"github.com/limeschool/gin"
	"io"
	"ps-go/tools"
	"strings"
)

type validate struct {
	request Request
}

type Validate interface {
	Bind(ctx *gin.Context) (map[string]any, error)
}

// Bind 绑定参数并校验。
func (v *validate) Bind(ctx *gin.Context) (map[string]any, error) {
	var value any
	var exist bool

	// 绑定query
	queryMap := v.getQuery(ctx)
	for key, field := range v.request.Query {
		value, exist = queryMap[key]
		newVal, ignore, err := field.Validate(value, exist)
		if !ignore {
			if err != nil {
				return nil, fmt.Errorf("request.query.%v %v", key, err)
			}
			queryMap[key] = newVal
		}
	}

	// 绑定header
	headerMap := v.getHeader(ctx)
	for key, field := range v.request.Header {
		value, exist = headerMap[key]
		newVal, ignore, err := field.Validate(value, exist)
		if !ignore {
			if err != nil {
				return nil, fmt.Errorf("request.header.%v %v", key, err)
			}
			headerMap[key] = newVal
		}
	}

	// 绑定body
	bodyMap := v.getBody(ctx)
	for key, field := range v.request.Body {
		value, exist = bodyMap[key]
		newVal, ignore, err := field.Validate(value, exist)
		if !ignore {
			if err != nil {
				return nil, fmt.Errorf("request.body.%v %v", key, err)
			}
			bodyMap[key] = newVal
		}
	}

	return gin.H{
		"query":  queryMap,
		"body":   bodyMap,
		"header": headerMap,
	}, nil
}

// getQuery 获取query请求参数
func (v *validate) getQuery(ctx *gin.Context) map[string]any {
	resp := make(map[string]any)

	queryStr := ctx.Request.URL.RawQuery
	if queryStr == "" {
		return resp
	}

	queryStr = strings.ReplaceAll(queryStr, "%20", "")
	queryStr = strings.ReplaceAll(queryStr, "%22", `"`)

	querySlice := strings.Split(queryStr, "&")

	for _, val := range querySlice {
		query := strings.Split(val, "=")
		if len(query) < 2 {
			continue
		}
		resp[query[0]] = query[1]
	}

	return resp
}

// getBody 获取body请求参数
func (v *validate) getBody(ctx *gin.Context) map[string]any {
	resp := make(map[string]any)
	byteData, _ := io.ReadAll(ctx.Request.Body)

	if v.request.Type == "xml" {
		_ = xml.Unmarshal(byteData, (*tools.XmlResult)(&resp))
	}

	if v.request.Type == "json" {
		_ = json.Unmarshal(byteData, &resp)
	}

	return resp
}

// getBody 获取请求header参数
func (v *validate) getHeader(ctx *gin.Context) map[string]any {
	resp := make(map[string]any)

	if len(v.request.Header) == 0 {
		return resp
	}

	for key, _ := range v.request.Header {
		val := ctx.Request.Header.Get(key)
		if val != "" {
			resp[key] = val
		}
	}

	return resp
}
