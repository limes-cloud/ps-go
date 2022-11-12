package model

import (
	"fmt"
	"github.com/limeschool/gin"
	"ps-go/tools/hash"
)

type RunLog struct {
	gin.CreateModel
	Trx     string `json:"trx"`     //唯一请求id
	Method  string `json:"method"`  //请求的方法
	Path    string `json:"path"`    //请求的路径
	Version string `json:"version"` //规则版本
	LogID   string `json:"log_id"`  //日志id
	Msg     string `json:"msg"`     //详细日志
}

func (s RunLog) Table(trx string) string {
	return fmt.Sprintf("run_log_%v", s.ComputeIndex(trx))
}

func (s RunLog) ComputeIndex(trx string) string {
	return hash.GetHash().Get(trx)
}

func (s *RunLog) Create(ctx *gin.Context) error {
	return database(ctx).Table(s.Table(s.Trx)).Create(s).Error
}

func (s *RunLog) DeleteByTrx(ctx *gin.Context, trx string) error {
	return database(ctx).Table(s.Table(trx)).Delete(s, "trx = ?", trx).Error
}

func (s *RunLog) OneByTrx(ctx *gin.Context, trx string) error {
	return database(ctx).Table(s.Table(trx)).Where("trx = ?", trx).First(s).Error
}
