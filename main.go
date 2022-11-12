package main

import (
	"log"
	"ps-go/engine"
	"ps-go/rooter"
	"ps-go/tools/hash"
	"ps-go/tools/pool"
)

func main() {

	// 协程池初始化
	pool.Init()
	// 调度引擎初始化
	engine.Init()
	// 初始化hash一致性算法
	hash.Init()
	// api 初始化
	rg := rooter.Init()

	// 启动并监听端口
	if err := rg.Run(":8080"); err != nil {
		log.Fatalln(err)
	}
}
