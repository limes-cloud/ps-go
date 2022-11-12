package service

import (
	"github.com/jinzhu/copier"
	"github.com/limeschool/gin"
	"ps-go/errors"
	"ps-go/model"
	"ps-go/types"
)

func GetRule(ctx *gin.Context, in *types.GetRuleRequest) (model.Rule, error) {
	var err error
	rule := model.Rule{}
	if in.ID != 0 {
		err = rule.OneByID(ctx, in.ID)
	}

	if in.Name != "" {
		err = rule.OneByName(ctx, in.Name)
	}

	return rule, err
}

func PageRule(ctx *gin.Context, in *types.PageRuleRequest) ([]model.Rule, int64, error) {
	rule := model.Rule{}
	return rule.Page(ctx, in.Page, in.Count, in)
}

func AddRule(ctx *gin.Context, in *types.AddRuleRequest) error {
	rule := model.Rule{}
	if copier.Copy(&rule, in) != nil {
		return errors.AssignError
	}
	return rule.Create(ctx)
}

func SwitchVersionRule(ctx *gin.Context, in *types.SwitchVersionRuleRequest) error {
	rule := model.Rule{}
	if copier.Copy(&rule, in) != nil {
		return errors.AssignError
	}
	return rule.SwitchVersion(ctx)
}

func DeleteRule(ctx *gin.Context, in *types.DeleteRuleRequest) error {
	rule := model.Rule{}
	if copier.Copy(&rule, in) != nil {
		return errors.AssignError
	}
	return rule.DeleteByID(ctx)
}
