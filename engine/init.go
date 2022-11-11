package engine

import (
	"github.com/limeschool/gin"
	"sync"
)

type Engine interface {
	Store
	NewValidate(req Request) Validate
	NewRunner(*gin.Context, *Rule, RunStore) Runner
	NewRunStore() RunStore
}

var eg *engine

type engine struct {
	Store
}

var validatePool = sync.Pool{New: func() any {
	return &validate{}
}}

// NewValidate 创建验证器
func (engine) NewValidate(req Request) Validate {
	vd := validatePool.Get().(*validate)
	validatePool.Put(vd)

	vd.request = req
	return vd
}

var runnerPool = sync.Pool{New: func() any {
	return &runner{}
}}

// NewRunStore 创建运行存储器
func (e engine) NewRunStore() RunStore {
	return &runStore{
		data: map[string]any{},
		lock: sync.RWMutex{},
	}
}

// NewRunner 创建运行调度器
func (e engine) NewRunner(ctx *gin.Context, rule *Rule, rStore RunStore) Runner {
	run := runnerPool.Get().(*runner)
	runnerPool.Put(run)

	run.rule = rule
	run.count = len(rule.Components)
	run.index = 0
	run.runStore = rStore
	run.wg = &sync.WaitGroup{}
	run.store = e.Store
	run.response = &responseChan{
		response: make(chan responseData),
		lock:     sync.RWMutex{},
	}
	run.ctx = ctx
	run.err = &errorChan{
		err:  make(chan error),
		lock: sync.RWMutex{},
	}

	// 初始化日志
	run.NewLogger()

	return run
}

// Init 初始化调度引擎
func Init() {
	eg = &engine{
		Store: NewStore(),
	}
}

// Get 获取调度引擎实例
func Get() Engine {
	if eg == nil {
		Init()
	}
	return eg
}
