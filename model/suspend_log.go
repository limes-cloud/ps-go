package model

import (
	"github.com/limeschool/gin"
	"gorm.io/gorm"
)

type SuspendLog struct {
	gin.CreateModel
	Trx          string `json:"trx"`           //唯一请求id
	Method       string `json:"method"`        //请求的方法
	Path         string `json:"path"`          //请求的路径
	Version      string `json:"version"`       //规则版本
	LogID        string `json:"log_id"`        //日志id
	Step         int    `json:"step"`          //总步数
	CurStep      int    `json:"cur_step"`      //当前执行步
	ErrCode      string `json:"err_code"`      //错误码
	ErrMsg       string `json:"err_msg"`       //错误原因
	Rule         string `json:"rule"`          //流程规则 map
	Data         string `json:"data"`          //流程上下文数据 map
	ErrComponent string `json:"err_component"` //错误组件名 slice
}

func (s SuspendLog) Table() string {
	return "suspend_log"
}

func (s *SuspendLog) Create(ctx *gin.Context) error {
	return database(ctx).Table(s.Table()).Create(s).Error
}

func (s *SuspendLog) DeleteByTrx(ctx *gin.Context, trx string) error {
	db := database(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(s.Table()).Delete(s, "trx = ?", trx).Error; err != nil {
			return err
		}
		log := RunLog{}
		return tx.Table(log.Table(trx)).Delete(&log, "trx = ?", trx).Error
	})

}

func (s *SuspendLog) DeleteByID(ctx *gin.Context, id int64) error {
	return database(ctx).Table(s.Table()).Delete(s, "id = ?", id).Error
}

func (s *SuspendLog) OneByTrx(ctx *gin.Context, trx string) error {
	return database(ctx).Table(s.Table()).Where("trx = ?", trx).First(s).Error
}

func (s *SuspendLog) OneByID(ctx *gin.Context, id int64) error {
	return database(ctx).Table(s.Table()).Where("id = ?", id).First(s).Error
}

func (s *SuspendLog) Update(ctx *gin.Context) error {
	return database(ctx).Table(s.Table()).Updates(s).Error
}

// Page 查询分页数据
func (s *SuspendLog) Page(ctx *gin.Context, page, count int, m interface{}, fs ...callback) ([]SuspendLog, int64, error) {
	var list []SuspendLog
	var total int64

	db := database(ctx).Table(s.Table()).
		Select("id,trx,method,path,version,log_id,step,cur_step,err_msg,err_component,created_at")

	db = gin.GormWhere(db, s.Table(), m)
	db = exec(db, fs...)

	if err := db.Count(&total).Error; err != nil {
		return nil, total, err
	}

	if err := db.Order("created_at desc").Offset((page - 1) * count).Limit(count).Find(&list).Error; err != nil {
		return list, total, err
	}

	return list, total, nil
}
