package service

import (
	"github.com/limeschool/gin"
	"ps-go/model"
	"ps-go/types"
)

func GetRunLog(ctx *gin.Context, in *types.GetRunLogRequest) (model.RunLog, error) {
	suspend := model.RunLog{}
	return suspend, suspend.OneByTrx(ctx, in.Trx)
}
