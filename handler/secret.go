package handler

import (
	"github.com/limeschool/gin"
	"ps-go/errors"
	"ps-go/service"
	"ps-go/types"
)

func GetSecret(ctx *gin.Context) {
	in := types.GetSecretRequest{}

	if ctx.ShouldBind(&in) != nil {
		ctx.RespError(errors.ParamsError)
		return
	}

	if in.ID == 0 && in.Name == "" {
		ctx.RespError(errors.ParamsError)
		return
	}

	if resp, err := service.GetSecret(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespData(resp)
	}
}

func PageSecret(ctx *gin.Context) {
	in := types.PageSecretRequest{}

	if ctx.ShouldBind(&in) != nil {
		ctx.RespError(errors.ParamsError)
		return
	}

	if resp, total, err := service.PageSecret(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespList(in.Page, in.Count, int(total), resp)
	}
}

func AddSecret(ctx *gin.Context) {
	in := types.AddSecretRequest{}
	if err := ctx.ShouldBind(&in); err != nil {
		ctx.RespError(errors.ParamsError)
		return
	}
	if err := service.AddSecret(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespSuccess()
	}
}

func UpdateSecret(ctx *gin.Context) {
	in := types.UpdateSecretRequest{}
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.RespError(errors.ParamsError)
		return
	}
	if err := service.UpdateSecret(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespSuccess()
	}
}

func DeleteSecret(ctx *gin.Context) {
	in := types.DeleteSecretRequest{}
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.RespError(errors.ParamsError)
		return
	}
	if err := service.DeleteSecret(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespSuccess()
	}
}
