package handler

import (
	"github.com/limeschool/gin"
	"ps-go/errors"
	"ps-go/service"
	"ps-go/types"
)

func GetRule(ctx *gin.Context) {
	in := types.GetRuleRequest{}

	if ctx.ShouldBind(&in) != nil {
		ctx.RespError(errors.ParamsError)
		return
	}

	if in.ID == 0 && in.Name == "" && in.Version == "" {
		ctx.RespError(errors.ParamsError)
		return
	}

	if resp, err := service.GetRule(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespData(resp)
	}
}

func PageRule(ctx *gin.Context) {
	in := types.PageRuleRequest{}

	if ctx.ShouldBind(&in) != nil {
		ctx.RespError(errors.ParamsError)
		return
	}

	if resp, total, err := service.PageRule(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespList(in.Page, in.Count, int(total), resp)
	}
}

func AddRule(ctx *gin.Context) {
	in := types.AddRuleRequest{}
	if err := ctx.ShouldBind(&in); err != nil {
		ctx.RespError(errors.ParamsError)
		return
	}
	if err := service.AddRule(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespSuccess()
	}
}

func SwitchRuleVersion(ctx *gin.Context) {
	in := types.SwitchVersionRuleRequest{}
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.RespError(errors.ParamsError)
		return
	}
	if err := service.SwitchVersionRule(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespSuccess()
	}
}

func DeleteRule(ctx *gin.Context) {
	in := types.DeleteRuleRequest{}
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.RespError(errors.ParamsError)
		return
	}
	if err := service.DeleteRule(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespSuccess()
	}
}
