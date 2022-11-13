package engine

import (
	"github.com/limeschool/gin"
	"ps-go/errors"
	"ps-go/model"
)

type store struct {
}

func NewStore() *store {
	return &store{}
}

type Store interface {
	LoadRule(ctx *gin.Context, method, path string) (*Rule, error)
	LoadScript(ctx *gin.Context, name string) (string, string, error)
}

// LoadRule 获取指定规则
func (s *store) LoadRule(ctx *gin.Context, method, path string) (*Rule, error) {
	rule := model.Rule{}
	if err := rule.OneByNameMethod(ctx, path, method); err != nil {
		return nil, errors.NewF("不存在流程：%v->%v", ctx.Request.Method, path)
	}

	er := Rule{Version: rule.Version}
	return &er, json.Unmarshal([]byte(rule.Rule), &er)
}

// LoadScript 获取指定脚本
func (s *store) LoadScript(ctx *gin.Context, name string) (string, string, error) {
	rule := model.Script{}

	if err := rule.OneByName(ctx, name); err != nil {
		return "", "", errors.NewF("加载脚本%v失败：%v", name, err.Error())
	}

	return rule.Script, rule.Version, nil
}
