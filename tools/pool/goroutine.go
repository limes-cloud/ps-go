package pool

import (
	"github.com/panjf2000/ants"
	"ps-go/consts"
)

type Task interface {
	Run()
}

var pool *ants.PoolWithFunc

// Init
//
//  @Description: 初始化协程池
func Init() {
	var err error

	pool, err = ants.NewUltimatePoolWithFunc(consts.GoRoutineCount, consts.GoRoutineExecSecond, func(in interface{}) {
		if task, ok := in.(Task); ok {
			task.Run()
		}
	}, false)

	if err != nil {
		panic("协程池初始化失败：" + err.Error())
	}
}

// Get
//
//  @Description: 获取协程池实例
//  @return *ants.PoolWithFunc
func Get() *ants.PoolWithFunc {
	return pool
}
