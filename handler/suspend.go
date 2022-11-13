package handler

import (
	"github.com/limeschool/gin"
	"ps-go/errors"
	"ps-go/service"
	"ps-go/types"
)

func GetSuspend(ctx *gin.Context) {
	in := types.GetSuspendRequest{}

	if ctx.ShouldBind(&in) != nil {
		ctx.RespError(errors.ParamsError)
		return
	}

	if in.ID == 0 && in.Trx == "" {
		ctx.RespError(errors.ParamsError)
		return
	}

	if resp, err := service.GetSuspend(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespData(resp)
	}
}

func PageSuspend(ctx *gin.Context) {
	in := types.PageSuspendRequest{}

	if ctx.ShouldBind(&in) != nil {
		ctx.RespError(errors.ParamsError)
		return
	}

	if resp, total, err := service.PageSuspend(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespList(in.Page, in.Count, int(total), resp)
	}
}

func SuspendRecover(ctx *gin.Context) {
	in := types.SuspendRecoverRequest{}
	if err := ctx.ShouldBind(&in); err != nil {
		ctx.RespError(errors.ParamsError)
		return
	}
	if data, err := service.SuspendRecover(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespJson(data)
	}
}

func UpdateSuspend(ctx *gin.Context) {
	in := types.UpdateSuspendRequest{}
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.RespError(errors.ParamsError)
		return
	}
	if err := service.UpdateSuspend(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespSuccess()
	}
}
