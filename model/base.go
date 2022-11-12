package model

import (
	"github.com/go-redis/redis/v8"
	"github.com/limeschool/gin"
	"gorm.io/gorm"
	"ps-go/consts"
	"ps-go/errors"
	"time"
)

type callback func(db *gorm.DB) *gorm.DB

func database(ctx *gin.Context) *gorm.DB {
	return ctx.Mysql(consts.ProcessScheduleDB)
}

func cache(ctx *gin.Context) *redis.Client {
	return ctx.Redis(consts.ProcessScheduleCache)
}

func delayDelCache(ctx *gin.Context, key string) {
	ctx.Redis(consts.ProcessScheduleCache).Del(ctx, key)
	go func() {
		time.Sleep(1 * time.Second)
		ctx.Redis(consts.ProcessScheduleCache).Del(ctx, key)
	}()
}

func exec(db *gorm.DB, fs ...callback) *gorm.DB {
	if fs != nil {
		for _, f := range fs {
			db = f(db)
		}
	}
	return db
}

func transferErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.DBNotFoundError
	} else {
		return errors.DBError
	}
}
