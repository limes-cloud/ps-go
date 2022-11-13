package handler

import (
	"github.com/limeschool/gin"
	"ps-go/errors"
	"ps-go/service"
	"ps-go/types"
)

func GetRunLog(ctx *gin.Context) {
	in := types.GetRunLogRequest{}

	if ctx.ShouldBind(&in) != nil {
		ctx.RespError(errors.ParamsError)
		return
	}

	if resp, err := service.GetRunLog(ctx, &in); err != nil {
		ctx.RespError(TransferError(err))
	} else {
		ctx.RespData(resp)
	}
}
