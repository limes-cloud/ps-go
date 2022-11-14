package handler

import (
	"github.com/limeschool/gin"
	"ps-go/errors"
	"ps-go/service"
	"ps-go/types"
)

func GetScript(ctx *gin.Context) {
	in := types.GetScriptRequest{}

	if ctx.ShouldBind(&in) != nil {
		ctx.RespError(errors.ParamsError)
		return
	}

	if in.ID == 0 && in.Name == "" {
		ctx.RespError(errors.ParamsError)
		return
	}

	if resp, err := service.GetScript(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespData(resp)
	}
}

func PageScript(ctx *gin.Context) {
	in := types.PageScriptRequest{}

	if ctx.ShouldBind(&in) != nil {
		ctx.RespError(errors.ParamsError)
		return
	}

	if resp, total, err := service.PageScript(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespList(in.Page, in.Count, int(total), resp)
	}
}

func AddScript(ctx *gin.Context) {
	in := types.AddScriptRequest{}
	if err := ctx.ShouldBind(&in); err != nil {
		ctx.RespError(errors.ParamsError)
		return
	}
	if err := service.AddScript(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespSuccess()
	}
}

func SwitchScriptVersion(ctx *gin.Context) {
	in := types.SwitchVersionScriptRequest{}
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.RespError(errors.ParamsError)
		return
	}
	if err := service.SwitchVersionScript(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespSuccess()
	}
}

func DeleteScript(ctx *gin.Context) {
	in := types.DeleteScriptRequest{}
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.RespError(errors.ParamsError)
		return
	}
	if err := service.DeleteScript(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespSuccess()
	}
}
