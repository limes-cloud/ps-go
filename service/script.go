package service

import (
	"github.com/jinzhu/copier"
	"github.com/limeschool/gin"
	"ps-go/errors"
	"ps-go/model"
	"ps-go/types"
)

func GetScript(ctx *gin.Context, in *types.GetScriptRequest) (model.Script, error) {
	var err error
	script := model.Script{}
	if in.ID != 0 {
		err = script.OneByID(ctx, in.ID)
	}

	if in.Name != "" {
		err = script.OneByName(ctx, in.Name)
	}

	return script, err
}

func PageScript(ctx *gin.Context, in *types.PageScriptRequest) ([]model.Script, int64, error) {
	script := model.Script{}
	return script.Page(ctx, in.Page, in.Count, in)
}

func AddScript(ctx *gin.Context, in *types.AddScriptRequest) error {
	script := model.Script{}
	if copier.Copy(&script, in) != nil {
		return errors.AssignError
	}
	return script.Create(ctx)
}

func UpdateScript(ctx *gin.Context, in *types.UpdateScriptRequest) error {
	script := model.Script{}
	if copier.Copy(&script, in) != nil {
		return errors.AssignError
	}
	return script.UpdateByID(ctx)
}

func DeleteScript(ctx *gin.Context, in *types.DeleteScriptRequest) error {
	script := model.Script{
		OperatorID: in.OperatorID,
		Operator:   in.Operator,
	}
	return script.DeleteByName(ctx, in.Name)
}
