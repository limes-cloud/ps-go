package engine

import (
	"github.com/limeschool/gin"
	"github.com/spf13/viper"
	"sync"
)

type Engine interface {
	Store
	NewValidate(req Request) Validate
	NewRunner(*gin.Context, *Rule, *viper.Viper) Runner
}

var eg *engine

type engine struct {
	*store
}

var validatePool = sync.Pool{New: func() any {
	return &validate{}
}}

func (engine) NewValidate(req Request) Validate {
	vd := validatePool.Get().(*validate)
	validatePool.Put(vd)

	vd.request = req
	return vd
}

var runnerPool = sync.Pool{New: func() any {
	return &runner{}
}}

func (e engine) NewRunner(ctx *gin.Context, rule *Rule, data *viper.Viper) Runner {
	vd := runnerPool.Get().(*runner)
	runnerPool.Put(vd)

	vd.rule = rule
	vd.count = len(rule.Components)
	vd.index = 0
	vd.runStore = &runStore{
		data: data,
	}
	vd.wg = &sync.WaitGroup{}
	vd.store = e.store
	vd.response = &responseChan{
		response: make(chan responseData),
		lock:     sync.RWMutex{},
	}
	vd.ctx = ctx
	vd.err = &errorChan{
		err:  make(chan error),
		lock: sync.RWMutex{},
	}

	return vd
}

// Init 初始化调度引擎
func Init() {
	eg = &engine{
		store: NewStore(),
	}
}

// Get 获取调度引擎实例
func Get() Engine {
	if eg == nil {
		Init()
	}
	return eg
}
